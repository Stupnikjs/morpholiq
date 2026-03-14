package morpho

import (
	"math/big"
)

type OnChainPosition struct {
	supplyShares      big.Int
	borrowShares      big.Int
	collateralAssets  big.Int
	totalSupplyAssets big.Int
	totalSupplyShares big.Int
	totalBorrowAssets big.Int
	totalBorrowShares big.Int
	oraclePrice       big.Int
}

// refactor pour faire un gros batch avec les / // pos par market
/*
func (e *Scanner) OnChainBatch() error {
	calls := []eth.CallFunc{}
	for _, sh := range e.WatchList.shards {
		for _, p := range sh.positions {
			pos := OnChainPosition{}
			calls = append(calls, []eth.CallFunc{
				eth.CallFunc(MorphoMain, PositionFunc, bor.MarketID, bor.Address).Returns(&p.supplyShares, &p.borrowShares, &p.collateralAssets),
				eth.CallFunc(MorphoMain, MarketFunc, bor.MarketID).Returns(&p.totalSupplyAssets, &p.totalSupplyShares, &p.totalBorrowAssets, &p.totalBorrowShares, new(big.Int), new(big.Int)),
				eth.CallFunc(oracleAddress, OraclePriceFunc).Returns(&p.oraclePrice),
			})
		}
	}
	err := e.Client.Call()
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}
	return &p, nil
}

// scaled by 10e6
func (p *BorrowPosition) HealthFactorUSD() *big.Int {
	if p.BorrowAssets == nil || p.BorrowAssets.Sign() == 0 ||
		p.BorrowAssetsUSD == nil || p.BorrowAssetsUSD.Sign() == 0 {
		return nil
	}

	if p.CollateralAssets == nil || p.CollateralAssetsUSD == nil ||
		p.CollateralAssets.Sign() == 0 || p.CollateralAssetsUSD.Sign() == 0 {
		return nil
	}

	num := new(big.Int).Mul(p.CollateralAssetsUSD, TenPowInt(6))
	return new(big.Int).Quo(num, p.BorrowAssetsUSD)

}

*/

// oracle_price c'est le prix du collateral en loan token
func HealthFactorOraclePrice(oraclePrice, borrowAssets, collateralAssets *big.Int) *big.Int {
	// HF = coll * oracle / borrowassets * oracle scale
	E36 := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	num := new(big.Int).Mul(collateralAssets, oraclePrice) // collateral * oraclePrice
	num.Mul(num, TenPowInt(6))                             // × 1e6 pour garder la précision
	denom := new(big.Int).Mul(borrowAssets, E36)           // borrowAssets * 1e36
	return new(big.Int).Div(num, denom)
}

// HF_lltv = hf × lltv / 1e18
// still scaled by 10e6
func HealthFactorLLTVScaled(hf, lltv *big.Int) *big.Int {

	if hf == nil || lltv == nil {
		return nil
	}
	return new(big.Int).Div(
		new(big.Int).Mul(hf, lltv),
		E18,
	)

}
