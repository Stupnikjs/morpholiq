package morpho

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

type Scanner struct {
	ClientHttp    *w3.Client
	ClientWs      *w3.Client
	Markets       []MorphoMarketParams
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
	return &Scanner{
		ClientHttp:    client,
		ClientWs:      clientWs,
		Markets:       markets,
		PositionCache: NewPositionCache(markets),
		oracleCh:      make(chan *types.Log, 100),
		positionCh:    make(chan *types.Log, 100),
	}
}

func NewPositionCache(markets []MorphoMarketParams) *PositionCache {
	bigMap := make(map[[32]byte]MarketCache, len(markets))

	for _, m := range markets {
		cache := make(map[common.Address]*BorrowPosition)
		bigMap[m.ID] = MarketCache{
			Oracle: m.Oracle,
			C:      cache,
		}
	}
	return &PositionCache{
		m: bigMap,
	}
}

// watchOraclePrices écoute les UpdatedPriceData (ou AnswerUpdated selon ton oracle)
func (e *Scanner) WatchOraclePrices(ctx context.Context) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{ORACLE_ADDRESS},
		Topics:    [][]common.Hash{{ORACLE_PRICE_UPDATED_TOPIC}},
	}

	sub, err := e.ClientWs.Subscribe(eth.NewLogs(e.oracleCh, query))
	if err != nil {
		panic(err)
	}

	for {
		select {
		case err := <-sub.Err():
			// reconnexion ou log d'erreur
			_ = err
			return
		case log := <-e.oracleCh:
   
			e.oracleCh <- log
		case <-ctx.Done():
			return
		}
	}
}

func (e *Scanner) Scan() error {

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
		for o := range e.oracleCh {
			ProcessOracleChange(o)
		}

		for ev := range e.positionCh {
			ProcessPositionChange(ev)
		}
	}
}

func (e *Scanner) WatchOraclePrices(ctx context.Context) {
	oracleAddresses := make([]common.Address, len(e.Markets))
	for i, m := range e.Markets {
		oracleAddresses[i] = m.Oracle
	}

	query := ethereum.FilterQuery{
		Addresses: oracleAddresses,
		Topics:    [][]common.Hash{{ORACLE_PRICE_UPDATED_TOPIC}},
	}

	// eth.NewLogs pousse directement dans oracleCh — pas besoin de relire
	sub, err := e.ClientWs.Subscribe(eth.NewLogs(e.oracleCh, query))
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			fmt.Println("oracle sub error:", err)
			return
		case <-ctx.Done():
			return
			// Ne pas lire oracleCh ici — Scan() s'en charge
		}
	}
}

func (e *Scanner) WatchPositions(ctx context.Context) {

	query := ethereum.FilterQuery{
		Addresses: []common.Address{MorphoMain},
		Topics: [][]common.Hash{{
			EventBorrow.Topic0, EventLiquidate.Topic0, EventRepay.Topic0,
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
			return
		case <-ctx.Done():
			return
		}
	}
}
func (e *Scanner) reconnectWs() {
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
