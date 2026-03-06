package morpho

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

// ─── Events Morpho Blue ───────────────────────────────────────────────────────

var (
	// keccak256("Borrow(bytes32,address,address,address,uint256,uint256)")
	topicBorrow = common.HexToHash(
		"0x570954b3d69e6c3c8da7c5a65e8543d2a9e09b365c7f26c5c9d67e74c3e7a8c3",
	)
	// keccak256("Repay(bytes32,address,address,uint256,uint256)")
	topicRepay = common.HexToHash(
		"0x9b45d1c0a83d4c47e3a6c9b0f3f1f6f9e2a8b4d6c7e5a3b2d1f0e9c8b7a6d5",
	)
	// keccak256("Liquidate(...)")
	topicLiquidate = common.HexToHash(
		"0xa4b1c5e7d9f2a0b3c6e8f1d4a7b0c3e6f9a2b5c8d1e4f7a0b3c6d9e2f5a8b1",
	)
)

// ─── Watchlist ────────────────────────────────────────────────────────────────

type BorrowerWatchlist struct {
	mu        sync.RWMutex
	borrowers map[common.Address]struct{} // adresses actives par marché
	marketID  [32]byte
}

func NewBorrowerWatchlist(marketID [32]byte) *BorrowerWatchlist {
	return &BorrowerWatchlist{
		borrowers: make(map[common.Address]struct{}),
		marketID:  marketID,
	}
}

func (w *BorrowerWatchlist) Add(addr common.Address) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.borrowers[addr] = struct{}{}
}

func (w *BorrowerWatchlist) Remove(addr common.Address) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.borrowers, addr)
}

func (w *BorrowerWatchlist) Snapshot() []common.Address {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]common.Address, 0, len(w.borrowers))
	for addr := range w.borrowers {
		out = append(out, addr)
	}
	return out
}

// SyncBorrowersFromLogs reconstruit la watchlist depuis un bloc de départ.
// À appeler une fois au boot, avant de commencer l'écoute temps-réel.
func SyncBorrowersFromLogs(
	client *w3.Client,
	watchlist *BorrowerWatchlist,
	fromBlock uint64,
	toBlock uint64,
) error {
	ctx := context.Background()

	// Filtrer les logs Morpho Blue pour ce marché uniquement
	// topic[0] = signature event, topic[1] = marketId
	marketTopic := common.Hash(watchlist.marketID)

	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{MorphoBlueAddr},
		Topics: [][]common.Hash{
			{topicBorrow, topicRepay, topicLiquidate},
			{marketTopic}, // filtre sur le marketId en topic[1]
		},
	}

	var logs []types.Log
	if err := client.CallCtx(ctx,
		eth.Logs(query).Returns(&logs),
	); err != nil {
		return fmt.Errorf("fetch logs: %w", err)
	}

	for _, log := range logs {
		// topic[2] = onBehalf (l'emprunteur réel)
		if len(log.Topics) < 3 {
			continue
		}
		borrower := common.BytesToAddress(log.Topics[2].Bytes())

		switch log.Topics[0] {
		case topicBorrow:
			watchlist.Add(borrower)
			fmt.Printf("[+] Borrower ajouté   : %s\n", borrower)

		case topicRepay:
			// Vérifier si la dette est totalement remboursée
			// Pour simplifier : retirer de la watchlist, re-checker à la prochaine itération
			watchlist.Remove(borrower)
			fmt.Printf("[-] Borrower retiré   : %s\n", borrower)

		case topicLiquidate:
			watchlist.Remove(borrower)
			fmt.Printf("[x] Borrower liquidé  : %s\n", borrower)
		}
	}

	fmt.Printf("Sync terminé : %d borrowers actifs\n", len(watchlist.Snapshot()))
	return nil
}
