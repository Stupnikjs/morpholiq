package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Stupnikjs/morpholiq/morpho"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

var (
	dRPC   = "https://lb.drpc.live/ethereum/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	pubRPC = "https://ethereum-rpc.publicnode.com"
	// wstETH (collateral) / USDC (loan) — LLTV 86% — un des marchés les plus actifs
	TestMarketID = [32]byte(
		common.HexToHash("0xb323495f7e4148be5643a4ea4a8221eef163e4bccfdedc2a6f4696baacbc86cc"),
	)
	BaseWETHUSDC = [32]byte(
		common.HexToHash("0x3b3769cfca57be2eaed03fcc5299c25691b77781a1e124e7a8d520eb9a7eabb5"),
	)
)

func main() {
	// Connexion au noeud Ethereum (remplace par ton RPC)
	client, err := w3.Dial(pubRPC)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	var latestBlock *big.Int
	if err := client.CallCtx(context.Background(),
		eth.BlockNumber().Returns(&latestBlock),
	); err != nil {
		panic(err)
	}

	// 2. Construire la watchlist
	watchList := morpho.NewBorrowerWatchlist(TestMarketID)

	// 3. Sync depuis le déploiement de Morpho Blue sur Base
	// Morpho Blue sur Base mainnet

	if err := morpho.SyncBorrowersFromLogs(
		client,
		watchList,
		latestBlock.Uint64()-50000, // fromBlock : début de l'historique
		latestBlock.Uint64(),       // toBlock   : maintenant
	); err != nil {
		panic(err)
	}

	fmt.Printf("Watchlist prête : %d borrowers\n", len(watchList.Snapshot()))

}
