package morpho

import (
	"fmt"
	"math/big"
	"time"

	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

type Scanner struct {
	Client    *w3.Client
	ApiCaller *MorphoApiCaller
	WatchList PositionStore
}

func NewScanner(client *w3.Client, markets []MorphoMarketParams) *Scanner {

	return &Scanner{
		Client:    client,
		ApiCaller: &MorphoApiCaller{Markets: markets},
		WatchList: *NewPositionStore(),
	}
}

func (e *Scanner) Refresh() error {
	bp, err := e.ApiCaller.FecthHotPosition(10)
	fmt.Println(len(bp))
	if err != nil {
		return err
	}
	for _, p := range bp {
		e.WatchList.Set(p.Address.String(), &p)
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

		e.WatchList.ForEach()
		// estimer tout les liquidables
		// onchainHF => EstimateProfit()

	}
}

// changer cette func pour update la borrowPosition
func (e *Scanner) OnChainCalc(pos BorrowPosition) (*big.Int, *big.Int, error) {
	var (
		supplyShares      big.Int
		borrowShares      big.Int
		collateralAssets  big.Int
		totalSupplyAssets big.Int
		totalSupplyShares big.Int
		totalBorrowAssets big.Int
		totalBorrowShares big.Int
		oraclePrice       big.Int
	)
	// get oracle for price by marketID

	err := e.Client.Call(
		eth.CallFunc(MorphoMain, PositionFunc, pos.MarketID, pos.Address).Returns(&supplyShares, &borrowShares, &collateralAssets),
		eth.CallFunc(MorphoMain, MarketFunc, pos.MarketID).Returns(&totalSupplyAssets, &totalSupplyShares, &totalBorrowAssets, &totalBorrowShares, new(big.Int), new(big.Int)),
		eth.CallFunc(h.MarketMap[pos.MarketID].Oracle, OraclePriceFunc).Returns(&oraclePrice),
	)
	if err != nil {
		fmt.Println("err:", err)
		return nil, nil, err
	}
	threshold := big.NewInt(1_000_000)
	borrowAssets := new(big.Int).Div(
		new(big.Int).Mul(
			&borrowShares,
			new(big.Int).Add(&totalBorrowAssets, big.NewInt(1)),
		),
		new(big.Int).Add(&totalBorrowShares, big.NewInt(1_000_000)),
	)
	hf := HealthFactorOraclePrice(&oraclePrice, borrowAssets, &collateralAssets)
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
		new(big.Int).Mul(&collateralAssets, &oraclePrice),
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
