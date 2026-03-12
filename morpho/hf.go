package morpho

import (
	"math/big"
)

var E18 *big.Int = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

// scaled by 10e6
func HealthFactor(p HFparams) *big.Int {
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

// HF_lltv = hf × lltv / 1e18
// still scaled by 10e6
func HealthFactorLLTVScaled(hf, lltv *big.Int) *big.Int {

	return new(big.Int).Div(
		new(big.Int).Mul(hf, lltv),
		E18,
	)
}

func (h *HFManager) GetLiquidable() []BorrowPosition {
	liquidable := []BorrowPosition{}
	threshold := big.NewInt(1_000_000) // 1.0 × 1e6

	for k, v := range h.HFMap {
		if v == nil || v.Sign() == 0 {
			continue // bad debt, skip
		}
		if v.Cmp(threshold) < 0 {
			continue // bad debt (collateral < borrow)
		}

		hfLltv := HealthFactorLLTVScaled(v, h.LLTVmap[k.MarketID])
		if hfLltv == nil || hfLltv.Sign() == 0 {
			continue
		}

		if hfLltv.Cmp(threshold) < 0 {
			liquidable = append(liquidable, k) // ✅ liquidable rentable
		}
	}
	// a partir de toutes les addresses on recalcule les HF onchain
	return liquidable
}
