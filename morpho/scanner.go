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
	bigMap := make(map[[32]byte]*MarketCache, len(markets))

	for _, m := range markets {
		cache := make(map[common.Address]*BorrowPosition)
		bigMap[m.ID] = &MarketCache{
			Mu:     sync.Mutex{},
			Oracle: m.Oracle,
			LLTV:   m.LLTV,
			C:      cache,
		}
	}
	return &PositionCache{
		m: bigMap,
	}
}

func (e *Scanner) Scan() error {
	e.RefreshCache(30)
	go e.WatchPositions(context.Background())
	go e.WatchOraclePrices(context.Background())
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.RefreshCache(30)
		}
	}()
	for {
		time.Sleep(1 * time.Second)

		// listen changement de prix Oracle ou Event position

		log := <-e.positionCh
		e.ProcessLog(log)

	}
}

func (e *Scanner) WatchOraclePrices(ctx context.Context) error {
	seen := make(map[common.Address]struct{})
	var oracles []common.Address
	for _, m := range e.Markets {
		if _, ok := seen[m.Oracle]; !ok {
			seen[m.Oracle] = struct{}{}
			oracles = append(oracles, m.Oracle)
		}
	}

	prices := make([]*big.Int, len(oracles))
	for i := range prices {
		prices[i] = new(big.Int)
	}

	// w3 batch : tous les calls partent en une seule requête JSON-RPC
	calls := make([]w3types.RPCCaller, len(oracles))
	for i, addr := range oracles {
		calls[i] = eth.CallFunc(addr, OraclePriceFunc).Returns(prices[i])
	}

	if err := e.ClientHttp.CallCtx(ctx, calls...); err != nil {
		return err
	}
	// 3. Batch call — 1 seul RPC
	e.OracleCache.Mu.Lock()
	defer e.OracleCache.Mu.Unlock()

	for i, addr := range oracles {
		data := &OracleData{
			Price: prices[i],
			Ts:    time.Now().Unix(),
		}
		e.OracleCache.C[addr] = data
	}
	return nil
}

// filtrer seulement les logs qui concernent //nos positions
func (e *Scanner) WatchPositions(ctx context.Context) {

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

func (e *Scanner) RefreshCache(n int) error {

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

// changer cette func pour update la borrowPosition
/*
func (e *Scanner) OnChainCalc(pos BorrowPosition) (*big.Int, *big.Int, error) {
	oracleAddress := common.HexToAddress("")
	p, err := e.GetsPosParams(&pos, oracleAddress)
	if err != nil {
		fmt.Println(err)
	}
	threshold := big.NewInt(1_000_000)
	borrowAssets := new(big.Int).Div(
		new(big.Int).Mul(
			&p.borrowShares,
			new(big.Int).Add(&p.totalBorrowAssets, big.NewInt(1)),
		),
		new(big.Int).Add(&p.totalBorrowShares, big.NewInt(1_000_000)),
	)
	hf := HealthFactorOraclePrice(&p.oraclePrice, borrowAssets, &p.collateralAssets)
	if hf.Cmp(threshold) < 0 {
		return hf, nil, nil
	}

	// incentive = 1e18/lltv - 1e18
	incentive := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Mul(E18, E18),     // 1e36
			h.MarketMap[pos.MarketID].LLTV, // 980000000000000000
		), // = 1.0204...e18
		E18, // - 1e18
	)

	if incentive.Cmp(MAX_INCENTIVE) > 0 {
		incentive = MAX_INCENTIVE
	}

	collateralInLoan := new(big.Int).Div(
		new(big.Int).Mul(&p.collateralAssets, &p.oraclePrice),
		TenPowInt(36),
	)

	// maxRepaid = collateralInLoan * 1e18 / (1e18 + incentive)
	maxRepaid := new(big.Int).Div(
		new(big.Int).Mul(collateralInLoan, E18),
		new(big.Int).Add(E18, incentive),
	)

	// on ne peut pas rembourser plus que la dette totale
	repaidDebt := borrowAssets
	if maxRepaid.Cmp(borrowAssets) < 0 {
		repaidDebt = maxRepaid
	}

	// seizedValue = (repaidDebt * oraclePrice / 1e36) * (1e18 + incentive) / 1e18
	// value du collateral recuperé

	x := new(big.Int).Mul(
		repaidDebt,
		new(big.Int).Add(E18, incentive),
	)

	seizedValue := new(big.Int).Div(x, TenPowInt(18))

	// profit = collatéral saisi - dette remboursée
	profit := new(big.Int).Sub(seizedValue, repaidDebt)

	return HealthFactorLLTVScaled(hf, h.MarketMap[pos.MarketID].LLTV), profit, nil

}

*/
