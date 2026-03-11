package morpho

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func HealthFactor(p HFparams) *big.Int {
	if p.borrowAssets == nil || p.borrowAssets.Sign() == 0 ||
		p.borrowAssetsUSD == nil || p.borrowAssetsUSD.Sign() == 0 {
		return nil
	}

	collateralDec := ExponentFloat(uint(p.collateralAssetDecimals))
	borrowDec := ExponentFloat(uint(p.borrowAssetDecimals))

	num := new(big.Float).SetPrec(128).SetInt(p.collateralAssets)
	num.Quo(num, collateralDec)         // wei → unité
	num.Mul(num, p.collateralAssetsUSD) // × prix                // × lltv normalisé

	den := new(big.Float).SetPrec(128).SetInt(p.borrowAssets)
	den.Quo(den, borrowDec)         // wei → unité
	den.Mul(den, p.borrowAssetsUSD) // × prix

	res := new(big.Float).Quo(num, den)
	res = res.Mul(res, ExponentFloat(9))

	resInt, _ := res.Int(new(big.Int))

	return resInt

}

// HF_lltv = hf × lltv / 1e18
func HealthFactorLLTVScaled(hf, lltv *big.Int) *big.Int {
	e18 := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	return new(big.Int).Div(
		new(big.Int).Mul(hf, lltv),
		e18,
	)
}

func (h *HFManager) GetLiquidable(lltv *big.Int) []common.Address {
	liquidable := []common.Address{}
	threshold := big.NewInt(1_000_000_000) // 1.0 × 1e9

	for k, v := range h.HFMap {
		if v == nil || v.Sign() == 0 {
			continue // bad debt, skip
		}
		if v.Cmp(threshold) < 0 {
			continue // bad debt (collateral < borrow)
		}
		hfLltv := HealthFactorLLTVScaled(v, lltv)
		if hfLltv == nil || hfLltv.Sign() == 0 {
			continue
		}
		if hfLltv.Cmp(threshold) < 0 {
			liquidable = append(liquidable, k.Address) // ✅ liquidable rentable
		}
	}
	return liquidable
}
