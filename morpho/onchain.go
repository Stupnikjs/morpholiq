package morpho

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3/module/eth"
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
func (e *Scanner) GetsPosParams(bor *BorrowPosition, oracleAddress common.Address) (*OnChainPosition, error) {
	p := OnChainPosition{}
	err := e.Client.Call(
		eth.CallFunc(MorphoMain, PositionFunc, bor.MarketID, bor.Address).Returns(&p.supplyShares, &p.borrowShares, &p.collateralAssets),
		eth.CallFunc(MorphoMain, MarketFunc, bor.MarketID).Returns(&p.totalSupplyAssets, &p.totalSupplyShares, &p.totalBorrowAssets, &p.totalBorrowShares, new(big.Int), new(big.Int)),
		eth.CallFunc(oracleAddress, OraclePriceFunc).Returns(&p.oraclePrice),
	)
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}
	return &p, nil
}

// scaled by 10e6
func HealthFactorUSD(p *ApiPosition) *big.Int {
	if p.borrowAssets == nil || p.borrowAssets.Sign() == 0 ||
		p.borrowAssetsUSD == nil || p.borrowAssetsUSD.Sign() == 0 {
		return nil
	}

	if p.collateralAssets == nil || p.collateralAssetsUSD == nil ||
		p.collateralAssets.Sign() == 0 || p.collateralAssetsUSD.Sign() == 0 {
		return nil
	}

	num := new(big.Int).Mul(p.collateralAssetsUSD, TenPowInt(6))
	return new(big.Int).Quo(num, p.borrowAssetsUSD)

}

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
