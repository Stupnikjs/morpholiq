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

	var params = []morpho.MorphoMarketParams{
		// wstUSD / USDC
		{
			ID:                      TestMarketID,
			ChainID:                 1, // mainnet
			LoanToken:               common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			CollateralToken:         common.HexToAddress("0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"),
			Oracle:                  common.HexToAddress("0x48F7E36EB6B826B2dF4B2E630B62Cd25e89E40e2"),
			IRM:                     common.HexToAddress("0x870aC11D48B15DB9a138Cf899d20F13F79Ba00BC"),
			LLTV:                    big.NewInt(860000000000000000),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
		},
	}

	bEngine := morpho.NewBorrowerEngine(params)
	err = bEngine.LoadBorrowerCache(params[0])

	if err != nil {
		fmt.Println(err)
	}

	liquidable := bEngine.GetLiquidableByMarketId(params[0])

	for _, l := range liquidable {
		fmt.Println(l.HealthFactor(params[0].CollateralTokenDecimals, params[0].LoanTokenDecimals))
	}
}
