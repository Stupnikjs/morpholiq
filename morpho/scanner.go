package morpho

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

type Scanner struct {
	wsRPC      string
	httpRPC    string
	ClientHttp *w3.Client
	ClientWs   *w3.Client
	oracleCh   chan *types.Log
	positionCh chan *types.Log
}

func NewScanner(rpc, websocket string) *Scanner {
	client, err := w3.Dial(rpc)

	if err != nil {
		panic(err)
	}
	clientWs, err := w3.Dial(websocket)
	if err != nil {
		panic(err)
	}
	return &Scanner{
		wsRPC:      websocket,
		httpRPC:    rpc,
		ClientHttp: client,
		ClientWs:   clientWs,
		oracleCh:   make(chan *types.Log, 100),
		positionCh: make(chan *types.Log, 100),
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

func (e *Scanner) Scan() error {
	cache := NewCache(MainnetParams)
	cache.ApiRefreshCache()
	go e.WatchPositions(context.Background())
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			//err := cache.ApiRefreshCache()
			// fmt.Println(err)
		}
	}()
	go func() {

		for {
			time.Sleep(3 * time.Second)
			err := cache.OnChainRefresh(e.ClientHttp)
			if err != nil {
				fmt.Println(err)
			}
			// decortiquer ces positions
			liq := cache.LiquidationPotential(6)
			fmt.Println(len(liq))
		}
	}()

	for {
		time.Sleep(1 * time.Second)

		// listen changement de prix Oracle ou Event position
		log := <-e.positionCh
		_ = log
		// fmt.Println(log)
		// e.ProcessLog(log)

	}
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

func (c *Scanner) ReconnectWs() {
	backoff := 1 * time.Second
	for {
		client, err := w3.Dial(c.wsRPC)
		if err == nil {
			c.ClientWs = client
			return
		}
		fmt.Printf("ws reconnect failed: %v, retry in %s\n", err, backoff)
		time.Sleep(backoff)
		backoff = min(backoff*2, 30*time.Second)
	}
}
