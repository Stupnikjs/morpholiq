package morpho

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

// health factor = (collateral × collateralPrice × LLTV) / (shares × sharePrice × borrowPrice)
// // Morpho SDK / BlueHelper
// assets = borrowShares * totalBorrowAssets / totalBorrowShares

type BorrowerStats struct {
	Shares              *big.Int // borrow shares
	BorrowAssets        *big.Int // valeur réelle empruntée
	BorrowAssetsUsd     *big.Float
	CollateralAssets    *big.Int   // collateral déposé
	CollateralAssetsUsd *big.Float // collateral déposé
	LLTV                *big.Int   // mettre ailleur peut etre

}

type BorrowerCache map[common.Address]BorrowerStats

type BorrowerEngine struct {
	// lecture sans lock, zéro contention
	snapshot atomic.Pointer[map[[32]byte]BorrowerCache]
}

func NewBorrowerEngine(params []MorphoMarketParams) *BorrowerEngine {
	engine := &BorrowerEngine{}

	initialMap := make(map[[32]byte]BorrowerCache, len(params))
	engine.snapshot.Store(&initialMap)

	return engine
}

func (b *BorrowerEngine) LoadBorrowerCache(param MorphoMarketParams) error {
	marketIDstr := "0x" + hex.EncodeToString(param.ID[:])
	cache := make(BorrowerCache, 1000)
	query := fmt.Sprintf(`{
        "query": "{ marketPositions(first: 1000, where: { marketUniqueKey_in: [\"%s\"], chainId_in: [%d] }) 
		{ items 
		    { user 
			     { address } 
				      state { borrowShares borrowAssets borrowAssetsUsd collateral collateralUsd } 
					   market { lltv }
					  } 
			     } 
		    }"
    }`, marketIDstr, uint(param.ChainID))

	resp, err := http.Post(
		"https://api.morpho.org/graphql",
		"application/json",
		strings.NewReader(query),
	)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	var result struct {
		Data struct {
			MarketPositions struct {
				Items []struct {
					User struct {
						Address string `json:"address"`
					} `json:"user"`
					State struct {
						BorrowShares    json.Number `json:"borrowShares"`
						BorrowAssets    json.Number `json:"borrowAssets"`
						BorrowAssetsUsd json.Number `json:"borrowAssetsUsd"`
						Collateral      json.Number `json:"collateral"`
						CollateralUsd   json.Number `json:"collateralUsd"`
					} `json:"state"`
					Market struct {
						LLTV json.Number `json:"lltv"`
					}
				} `json:"items"`
			} `json:"marketPositions"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	for _, item := range result.Data.MarketPositions.Items {
		// Ignorer les positions sans emprunt actif
		if item.State.BorrowShares == "0" || item.State.BorrowShares == "" {
			continue
		}

		cache[common.HexToAddress(item.User.Address)] = BorrowerStats{
			Shares:              ParseBigInt(item.State.BorrowShares.String()),
			BorrowAssets:        ParseBigInt(item.State.BorrowAssets.String()),
			BorrowAssetsUsd:     ParseBigFloat(item.State.BorrowAssetsUsd.String()),
			CollateralAssets:    ParseBigInt(item.State.Collateral.String()),
			CollateralAssetsUsd: ParseBigFloat(item.State.CollateralUsd.String()),
			LLTV:                ParseBigInt(item.Market.LLTV.String()),
		}

	}
	old := b.snapshot.Load()
	newMap := make(map[[32]byte]BorrowerCache, len(*old)+1)
	maps.Copy(newMap, *old)
	newMap[param.ID] = cache
	swapped := b.snapshot.CompareAndSwap(old, &newMap)
	if swapped {
		return nil
	}
	return fmt.Errorf("swap failed ")
}

func (e *BorrowerEngine) Get(marketID [32]byte) BorrowerCache {
	snapshot := e.snapshot.Load() // atomique
	return (*snapshot)[marketID]
}

// Écrire — copie + swap
func (e *BorrowerEngine) Update(marketID [32]byte, cache BorrowerCache) {
	for {
		old := e.snapshot.Load()

		// Copie
		newMap := make(map[[32]byte]BorrowerCache, len(*old))
		maps.Copy(newMap, *old)
		newMap[marketID] = cache

		// Swap atomique — CompareAndSwap pour éviter les races entre writers
		if e.snapshot.CompareAndSwap(old, &newMap) {
			return
		}
		// Si un autre writer a swappé entre temps → on recommence
	}
}
func (s *BorrowerStats) HealthFactor(colDecimals, borrowDecimals uint16) *big.Float {
	if s.BorrowAssets == nil || s.BorrowAssets.Sign() == 0 ||
		s.BorrowAssetsUsd == nil || s.BorrowAssetsUsd.Sign() == 0 {
		return nil
	}

	e18 := new(big.Float).SetPrec(128).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil),
	)
	// LLTV normalisé proprement
	lltv := new(big.Float).SetPrec(128).Quo(
		new(big.Float).SetInt(s.LLTV), e18,
	)

	collateralDec := new(big.Float).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(colDecimals)), nil),
	)
	borrowDec := new(big.Float).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(borrowDecimals)), nil),
	)

	num := new(big.Float).SetPrec(128).SetInt(s.CollateralAssets)
	num.Quo(num, collateralDec)         // wei → unité
	num.Mul(num, s.CollateralAssetsUsd) // × prix
	num.Mul(num, lltv)                  // × lltv normalisé

	den := new(big.Float).SetPrec(128).SetInt(s.BorrowAssets)
	den.Quo(den, borrowDec)         // wei → unité
	den.Mul(den, s.BorrowAssetsUsd) // × prix

	return new(big.Float).Quo(num, den)
}

func (e *BorrowerEngine) GetLiquidableByMarketId(param MorphoMarketParams) []BorrowerStats {

	cache := e.Get(param.ID)
	liquidable := make([]BorrowerStats, 0, len(cache))
	one := new(big.Float).SetFloat64(1.0)
	for _, v := range cache {
		hf := v.HealthFactor(param.CollateralTokenDecimals, param.LoanTokenDecimals)

		if hf == nil {
			continue
		}
		if hf.Cmp(one) < 0 {
			liquidable = append(liquidable, v)
		}
	}
	return liquidable
}
