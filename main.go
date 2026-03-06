package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	dRPC   = "https://lb.drpc.live/ethereum/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	pubRPC = ""
	// wstETH (collateral) / USDC (loan) — LLTV 86% — un des marchés les plus actifs
	TestMarketID = [32]byte(
		common.HexToHash("0xb323495f7e4148be5643a4ea4a8221eef163e4bccfdedc2a6f4696baacbc86cc"),
	)
)

func main() {
	// Connexion au noeud Ethereum (remplace par ton RPC)
	client, err := w3.Dial(dRPC)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	market, err := GetMorphoMarketState(client, TestMarketID)

	fmt.Println(market)

}
