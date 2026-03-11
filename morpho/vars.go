package morpho

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	DRPC   = "https://lb.drpc.live/ethereum/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	PubRPC = "https://ethereum-rpc.publicnode.com"
	// wstETH (collateral) / USDC (loan) — LLTV 86% — un des marchés les plus actifs
	TestMarketID = [32]byte(
		common.HexToHash("0xb323495f7e4148be5643a4ea4a8221eef163e4bccfdedc2a6f4696baacbc86cc"),
	)
	BaseWETHUSDC = [32]byte(
		common.HexToHash("0x3b3769cfca57be2eaed03fcc5299c25691b77781a1e124e7a8d520eb9a7eabb5"),
	)
	MorphoMain = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")

	Params = []MorphoMarketParams{
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
)
