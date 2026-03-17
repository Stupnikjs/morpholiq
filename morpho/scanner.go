package morpho

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

type Scanner struct {
	ClientHttp    *w3.Client
	ClientWs      *w3.Client
	Markets       []MorphoMarketParams
	OracleCache   *OracleCache
	PositionCache *PositionCache
	oracleCh      chan *types.Log
	positionCh    chan *types.Log
}

func NewScanner(markets []MorphoMarketParams) *Scanner {
	client, err := w3.Dial(BASEDRPC)

	if err != nil {
		panic(err)
	}
	clientWs, err := w3.Dial(BASEDRPCWS)
	if err != nil {
		panic(err)
	}
	return &Scanner{
		ClientHttp:    client,
		ClientWs:      clientWs,
		Markets:       markets,
		OracleCache:   NewOracleCache(markets),
		PositionCache: NewPositionCache(markets),
		oracleCh:      make(chan *types.Log, 100),
		positionCh:    make(chan *types.Log, 100),
	}
}

func NewPositionCache(markets []MorphoMarketParams) *PositionCache {
	bigMap := make(map[[32]byte]*Market, len(markets))

	for _, m := range markets {
		cache := make(map[common.Address]*BorrowPosition)
		bigMap[m.ID] = &Market{
			Mu: sync.RWMutex{},
			MarketCache: MarketCache{
				Oracle: m.Oracle,
				C:      cache,
			},
			MarketStats: MarketStats{},
		}
	}
	return &PositionCache{
		m: bigMap,
	}
}

/*







 */

func (e *Scanner) Scan() error {
	e.ApiRefreshCache(30)
	go e.WatchPositions(context.Background())
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.ApiRefreshCache(30)
		}
	}()
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			e.OnChainRefresh()
		}
	}()

	e.OnChainRefresh()
	// e.UpdateHF()
	for {
		time.Sleep(1 * time.Second)

		// listen changement de prix Oracle ou Event position
		log := <-e.positionCh
		e.ProcessLog(log)

	}
}

// filtrer seulement les logs qui concernent //nos positions
func (e *Scanner) WatchPositions(ctx context.Context) {
	fmt.Println("Watching Positions .. ")
	query := ethereum.FilterQuery{
		Addresses: []common.Address{MorphoMain},
		Topics: [][]common.Hash{{
			EventBorrow.Topic0, EventLiquidate.Topic0, EventAccrueInterest.Topic0,
		}},
	}

	sub, err := e.ClientWs.Subscribe(eth.NewLogs(e.positionCh, query))
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			fmt.Println("position sub error:", err)
			sub.Unsubscribe()
			e.ReconnectWs()
		case <-ctx.Done():
			return
		}
	}
}

func (e *Scanner) ApiRefreshCache(n int) error {

	for _, ma := range e.Markets {
		fetched, err := FecthBorrowersFromMarket(ma, n)
		if err != nil {
			return err
		}

		for _, p := range fetched {
			e.PositionCache.m[ma.ID].C[p.Address] = &p
		}

	}
	return nil
}

func (e *Scanner) OnChainRefresh() error {

	ctx := context.Background()
	var calls []w3types.RPCCaller

	marketMap, marketCalls := e.MarketStatsCalls()
	oraclePrice, oracleCalls := e.OracleCalls()

	calls = append(calls, marketCalls...)
	calls = append(calls, oracleCalls...)
	if err := e.ClientHttp.CallCtx(ctx, calls...); err != nil {
		return err
	}

	// update
	for id, m := range e.PositionCache.m {
		ms := marketMap[id]
		m.Mu.Lock()
		m.MarketStats.TotalBorrowAssets = ms.TotalBorrowAssets
		m.MarketStats.TotalBorrowShares = ms.TotalBorrowShares
		m.Mu.Unlock()
	}

	// unpack oracle
	for addr, p := range oraclePrice {

		e.OracleCache.Mu.Lock()
		data := &OracleData{
			Price: p,
			Ts:    time.Now().Unix(),
		}
		e.OracleCache.C[addr] = data
		e.OracleCache.Mu.Unlock()
	}
	return nil
}

func (e *Scanner) MarketStatsCalls() (map[[32]byte]*MarketStats, []w3types.RPCCaller) {
	var calls []w3types.RPCCaller

	marketStates := make(map[[32]byte]*MarketStats, len(e.Markets))

	for id := range e.PositionCache.m {
		ms := MarketStats{
			TotalBorrowAssets: new(big.Int),
			TotalBorrowShares: new(big.Int),
		}

		marketStates[id] = &ms
		calls = append(calls, eth.CallFunc(MorphoMain, MarketFunc, id).Returns(
			new(big.Int), new(big.Int), // supply on s'en fout
			ms.TotalBorrowAssets, ms.TotalBorrowShares,
			new(big.Int), new(big.Int),
		))
	}

	return marketStates, calls
}

// RETOURNE LES ORACLE CALLS AVEC LES POINTEURS DE RESULT
func (e *Scanner) OracleCalls() (map[common.Address]*big.Int, []w3types.RPCCaller) {
	oraclePrices := make(map[common.Address]*big.Int)
	var calls []w3types.RPCCaller

	for _, m := range e.PositionCache.m {
		if _, ok := oraclePrices[m.Oracle]; !ok {
			price := new(big.Int)
			oraclePrices[m.Oracle] = price
			calls = append(calls, eth.CallFunc(m.Oracle, OraclePriceFunc).Returns(price))
		}
	}
	return oraclePrices, calls
}

func (e *Scanner) ReconnectWs() {
	backoff := 1 * time.Second
	for {
		client, err := w3.Dial(BASEDRPCWS)
		if err == nil {
			e.ClientWs = client
			return
		}
		fmt.Printf("ws reconnect failed: %v, retry in %s\n", err, backoff)
		time.Sleep(backoff)
		backoff = min(backoff*2, 30*time.Second)
	}
}

func (e *Scanner) ProcessLog(log *types.Log) {
	switch log.Topics[0] {
	case EventAccrueInterest.Topic0:
		e.PositionCache.AccrueInterestEventProcess(log)
	case EventBorrow.Topic0:
		e.PositionCache.BorrowEventProcess(log)
	case EventRepay.Topic0:

	case EventLiquidate.Topic0:
		e.PositionCache.LiquidateEventProcess(log)
	}
}

func (e *Scanner) UpdateHF() {
	// loop over markets to update hf
}

// load Markets
