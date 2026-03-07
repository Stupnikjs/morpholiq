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

type BorrowerStats struct {
	Shares *big.Int
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
	cache := make(map[common.Address]BorrowerStats, 1000)
	query := fmt.Sprintf(`{
        "query": "{ marketPositions(first: 1000, where: { marketUniqueKey_in: [\"%s\"], chainId_in: [%d] }) { items { user { address } state { borrowShares } } } }"
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
						BorrowShares json.Number `json:"borrowShares"`
					} `json:"state"`
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
		shares, _ := item.State.BorrowShares.Int64()
		bigShares := big.NewInt(shares)
		cache[common.HexToAddress(item.User.Address)] = BorrowerStats{
			Shares: bigShares,
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
