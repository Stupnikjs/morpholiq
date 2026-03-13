package morpho

import (
	"maps"
	"math/big"
	"sync/atomic"
)

var E18 *big.Int = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

type HFManager struct {
	MarketMap MarketMap
	LLTVmap   map[[32]byte]*big.Int
	HFMap     atomic.Pointer[map[BorrowPosition]*big.Int]
}

// scaled by 10e6
type HFparams struct {
	borrowAssets, collateralAssets, borrowAssetsUSD, collateralAssetsUSD, LLTV *big.Int
	borrowAssetDecimals, collateralAssetDecimals                               uint16
}

type MarketMap map[[32]byte]MorphoMarketParams

func (h *HFManager) GetHF(pos BorrowPosition) *big.Int {
	return (*h.HFMap.Load())[pos]
}

// écriture
func (h *HFManager) SetHF(pos BorrowPosition, hf *big.Int) {
	for {
		old := h.HFMap.Load()
		newMap := make(map[BorrowPosition]*big.Int, len(*old))
		maps.Copy(newMap, *old)
		newMap[pos] = hf
		if h.HFMap.CompareAndSwap(old, &newMap) {
			return
		}
	}
}

// remplacement complet après refresh
func (h *HFManager) ReplaceHFMap(newMap map[BorrowPosition]*big.Int) {
	h.HFMap.Store(&newMap)
}

// scaled by 10e6
func HealthFactorUSD(p HFparams) *big.Int {
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

func (h *HFManager) GetLiquidable() []BorrowPosition {
	liquidable := []BorrowPosition{}
	threshold := big.NewInt(1_000_000) // 1.0 × 1e6

	for k, v := range *h.HFMap.Load() {
		if v == nil || v.Sign() == 0 {
			continue
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

/*
seizedCollateral = repaidDebt * oraclePrice / 1e36 * (1e18 + incentive) / 1e18

La liquidation est valide tant que :
seizedCollateral <= collateralAssets  // tu ne peux pas saisir plus que le collatéral dispo

*/
