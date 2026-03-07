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

func NewBorrowerEngine(size uint) *BorrowerEngine {
	engine := &BorrowerEngine{}

	initialMap := make(map[[32]byte]BorrowerCache, size)
	engine.snapshot.Store(&initialMap)

	return engine
}

func (b *BorrowerEngine) LoadBorrowerCache(marketID [32]byte, chainID int) error {
	marketIDstr := "0x" + hex.EncodeToString(marketID[:])
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
    }`, marketIDstr, chainID)

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
			BorrowAssetsUsd:     ParseBigFloat(item.State.CollateralUsd.String()),
			CollateralAssets:    ParseBigInt(item.State.Collateral.String()),
			CollateralAssetsUsd: ParseBigFloat(item.State.CollateralUsd.String()),
			LLTV:                ParseBigInt(item.Market.LLTV.String()),
		}

	}
	old := b.snapshot.Load()
	newMap := make(map[[32]byte]BorrowerCache, len(*old)+1)
	maps.Copy(newMap, *old)
	newMap[marketID] = cache
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
func (s *BorrowerStats) HealthFactor() *big.Float {
	if s.BorrowAssets == nil || s.BorrowAssets.Sign() == 0 {
		return nil
	}
	e18 := new(big.Float).SetPrec(128).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	)
	lltv := new(big.Float).Quo(
		new(big.Float).SetInt(s.LLTV),
		e18, // normalize 1e18 → [0,1]
	)

	// (collateral × collateralPrice × lltv)
	num := new(big.Float).SetInt(s.CollateralAssets)
	num.Mul(num, lltv)

	// (borrow × borrowPrice)
	den := new(big.Float).SetInt(s.BorrowAssets)
	den.Mul(den, s.BorrowAssetsUsd)

	return new(big.Float).Quo(num, den)
}

func (s *BorrowerStats) Print() {
	fmt.Println("Borrower stat ")
	fmt.Printf("borrow assets: %d \n", s.BorrowAssets.Int64())
	f, _ := s.BorrowAssetsUsd.Float64()
	fmt.Printf("borrow assetsUSD: %f \n", f)
	fmt.Printf("borrow shares: %d \n", s.Shares.Int64())
	fmt.Printf("collateral: %d \n", s.CollateralAssets.Int64())
	f, _ = s.CollateralAssetsUsd.Float64()
	fmt.Printf("collateralprice: %f \n", f)

}
