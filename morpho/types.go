package morpho

import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

type BorrowerStats struct {
	Shares           *big.Int // borrow shares
	BorrowAssets     *big.Int // valeur réelle empruntée
	CollateralAssets *big.Int // collateral déposé
	LLTV             *big.Int // mettre ailleur peut etre

}

type MarketState struct {
	MarketParams       MorphoMarketParams
	BorrowAssetUsd     *big.Float
	CollateralAssetUsd *big.Float
	BorrowerCache      BorrowerCache
}
type BorrowerCache map[common.Address]BorrowerStats

type MorphoEngine struct {
	// lecture sans lock, zéro contention
	snapshot atomic.Pointer[map[[32]byte]MarketState]
}

type HFIndex struct {
	// HF → []Address (trié naturellement par la map triée)
	byHF map[string][]common.Address // clé = HF.Text('f', 18) pour comparaison exacte

	// Address → HF (pour update O(1))
	byAddr map[common.Address]*big.Float

	// Slice triée des clés HF pour itération rapide
	sorted []string

	mu sync.RWMutex
}
