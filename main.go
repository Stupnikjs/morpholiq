package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Stupnikjs/morpholiq/morpho"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

func main() {
	// Connexion au noeud Ethereum (remplace par ton RPC)
	client, err := w3.Dial(morpho.PubRPC)
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

	m, err := morpho.FetchMarkets()
	fmt.Println(m[:10])

}
