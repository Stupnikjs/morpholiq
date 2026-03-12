package main

import (
	"context"
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

	engine := morpho.InitEngine(morpho.Params)
	err = engine.LoadBorrowerCache(morpho.Params[0])
	if err != nil {
		panic(err)
	}

}

/*

-- Call a l'api morpho pour initier une struct de suivi des liquidations futures // recall toute les 10min
-- tout les block reactualisation des HF
-- Si position liquidable ==> simulation de transaction / calcul du slipage ==> estimation des gains potentiels
-- envoie de la transaction

*/
