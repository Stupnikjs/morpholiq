package morpho

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// the point of borrowers.go is to filter morpho api borrowers to keep the liquidable ones
// more precise on chain checks will validate liquidate calls
// health factor = (collateral × collateralPrice × LLTV) / (shares × sharePrice × borrowPrice)
// // Morpho SDK / BlueHelper
// assets = borrowShares * totalBorrowAssets / totalBorrowShares

func NewMorphoEngine(params []MorphoMarketParams) *MorphoEngine {
	engine := &MorphoEngine{}

	initialMap := make(map[[32]byte]MarketState, len(params))
	engine.snapshot.Store(&initialMap)

	return engine
}

func (b *MorphoEngine) LoadBorrowerCache(param MorphoMarketParams) error {
	marketIDstr := "0x" + hex.EncodeToString(param.ID[:])
	marketState := MarketState{
		MarketParams:  param,
		BorrowerCache: make(BorrowerCache),
	}
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

		marketState.BorrowerCache[common.HexToAddress(item.User.Address)] = BorrowerStats{
			Shares:           ParseBigInt(item.State.BorrowShares.String()),
			BorrowAssets:     ParseBigInt(item.State.BorrowAssets.String()),
			CollateralAssets: ParseBigInt(item.State.Collateral.String()),
			LLTV:             ParseBigInt(item.Market.LLTV.String()),
		}
		if marketState.BorrowAssetUsd == nil {
			marketState.BorrowAssetUsd = ParseBigFloat(item.State.BorrowAssetsUsd.String())
		}
		if marketState.CollateralAssetUsd == nil {
			marketState.CollateralAssetUsd = ParseBigFloat(item.State.CollateralUsd.String())
		}
	}
	old := b.snapshot.Load()
	newMap := make(map[[32]byte]MarketState, len(*old)+1)
	maps.Copy(newMap, *old)
	newMap[param.ID] = marketState
	swapped := b.snapshot.CompareAndSwap(old, &newMap)
	if swapped {
		return nil
	}
	return fmt.Errorf("swap failed ")
}

func (e *MorphoEngine) Get(marketID [32]byte) MarketState {
	snapshot := e.snapshot.Load() // atomique
	return (*snapshot)[marketID]
}

// Écrire — copie + swap
func (e *MorphoEngine) Update(marketID [32]byte, state MarketState) {
	for {
		old := e.snapshot.Load()

		// Copie
		newMap := make(map[[32]byte]MarketState, len(*old))
		maps.Copy(newMap, *old)
		newMap[marketID] = state

		// Swap atomique — CompareAndSwap pour éviter les races entre writers
		if e.snapshot.CompareAndSwap(old, &newMap) {
			return
		}
		// Si un autre writer a swappé entre temps → on recommence
	}
}

// HF index is a struct of 2 maps indexing HF by common.Address
// enabling quick update of HF
func (e *MorphoEngine) BuildHFIndex() HFIndex {
	return HFIndex{}
}

/*
OLD HealthFactor

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




*/
