package morpho

import (
	"fmt"
	"maps"
	"math/big"
	"sync/atomic"

	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
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

func (e *MorphoEngine) NewHFManager() *HFManager {
	hfIndex := e.BuildHFIndex()
	manager := HFManager{}
	manager.HFMap.Store(&hfIndex)
	return &manager
}

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

func (h *HFManager) OnChainCalc(client *w3.Client, pos BorrowPosition) (*big.Int, *big.Int, error) {
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

	err := client.Call(
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

/*
seizedCollateral = repaidDebt * oraclePrice / 1e36 * (1e18 + incentive) / 1e18

La liquidation est valide tant que :
seizedCollateral <= collateralAssets  // tu ne peux pas saisir plus que le collatéral dispo

*/
