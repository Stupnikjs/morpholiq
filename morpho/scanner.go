package morpho

import (
	"context"
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
	ApiCaller     *MorphoApiCaller
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
		ApiCaller:     &MorphoApiCaller{Markets: markets},
		PositionCache: NewPositionCache(markets),
		oracleCh:      make(chan *types.Log, 100),
		positionCh:    make(chan *types.Log, 100),
	}
}

func NewPositionCache(markets []MorphoMarketParams) *PositionCache {
	bigMap := make(map[[32]byte]map[common.Address]*BorrowPosition, len(markets))
	for _, m := range markets {
		bigMap[m.ID] = make(map[common.Address]*BorrowPosition)
	}
	return &PositionCache{
		m: bigMap,
	}
}

// watchOraclePrices écoute les UpdatedPriceData (ou AnswerUpdated selon ton oracle)
func (e *Scanner) watchOraclePrices(ctx context.Context) {
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

func (e *Scanner) ApiCall() error {
	bp, err := e.ApiCaller.FecthHotPosition(10)
	if err != nil {
		return err
	}
	for _, p := range bp {
		e.WatchList.m[p.MarketID][p.Address] = &p
	}
	return nil

}

func (e *Scanner) Scan() error {

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.Refresh()
		}
	}()

	for {
		time.Sleep(1 * time.Second)

		// listen changement de prix Oracle ou Event position

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
