package morpho

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

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
						BorrowShares        json.Number `json:"borrowShares"`
						BorrowAssets        json.Number `json:"borrowAssets"`
						BorrowAssetsUsd     json.Number `json:"borrowAssetsUsd"`
						Collateral          json.Number `json:"collateral"`
						CollateralAssetsUsd json.Number `json:"collateralUsd"`
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
			Shares:              ParseBigInt(item.State.BorrowShares.String()),
			BorrowAssets:        ParseBigInt(item.State.BorrowAssets.String()),
			BorrowAssetsUSD:     ParseBigInt(item.State.BorrowAssetsUsd.String()),
			CollateralAssetsUSD: ParseBigInt(item.State.CollateralAssetsUsd.String()),
			CollateralAssets:    ParseBigInt(item.State.Collateral.String()),
			LLTV:                ParseBigInt(item.Market.LLTV.String()),
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

func (e *MorphoEngine) BuildHFIndex() map[BorrowPosition]*big.Int {
	hfMap := make(map[BorrowPosition]*big.Int)
	for _, ms := range *e.snapshot.Load() {
		ms.MergeHFInto(hfMap) // écrit directement dedans
	}
	return hfMap
}

func (m *MarketState) MergeHFInto(hfMap map[BorrowPosition]*big.Int) {
	for k, v := range m.BorrowerCache {
		pos := BorrowPosition{MarketID: m.MarketParams.ID, Address: k}
		hfParams := HFparams{
			borrowAssets:            v.BorrowAssets,
			borrowAssetsUSD:         v.BorrowAssetsUSD,
			collateralAssets:        v.CollateralAssets,
			collateralAssetsUSD:     v.CollateralAssetsUSD,
			borrowAssetDecimals:     m.MarketParams.LoanTokenDecimals,
			collateralAssetDecimals: m.MarketParams.CollateralTokenDecimals,
		}
		hfMap[pos] = HealthFactor(hfParams)
	}
}

func (e *MorphoEngine) DebugPosition(addr common.Address) {
	for _, ms := range *e.snapshot.Load() {
		v, ok := ms.BorrowerCache[addr]
		if !ok {
			continue
		}
		fmt.Printf("address:          %s\n", addr.Hex())
		fmt.Printf("collateralAssets: %s\n", v.CollateralAssets.String())
		fmt.Printf("borrowAssets:     %s\n", v.BorrowAssets.String())
		fmt.Printf("lltv:             %s\n", v.LLTV.String())
	}
}
