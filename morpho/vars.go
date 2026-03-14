package morpho

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	DRPC         = "https://lb.drpc.live/ethereum/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	BASEDRPC     = "https://lb.drpc.live/base/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	BASEDRPCWS   = "wss://lb.drpc.live/base/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	PubRPC       = "https://ethereum-rpc.publicnode.com"
	BaseWETHUSDC = [32]byte(
		common.HexToHash("0x3b3769cfca57be2eaed03fcc5299c25691b77781a1e124e7a8d520eb9a7eabb5"),
	)
	MorphoMain    = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")
	MAX_INCENTIVE = new(big.Int).SetUint64(150_000_000_000_000_000) // 0.15e18 = 15%
	MinUSDProfit  = 2
	BaseParams    = []MorphoMarketParams{
		// ─────────────────────────────────────────────────────────────
		// 1. cbBTC / USDC — 86% LLTV
		//    Le plus gros marché sur Base, ~$1B+ de borrow actif
		//    Powering Coinbase's BTC-backed loan product
		// ─────────────────────────────────────────────────────────────
		{
			ID: [32]byte(common.HexToHash(
				"0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836")),
			ChainID:         8453,
			LoanToken:       common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"), // USDC
			CollateralToken: common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"), // cbBTC
			Oracle:          common.HexToAddress("0x663BECd10daE6C4A3Dcd89F1d76c1174199639B9"),
			IRM:             common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687"), // AdaptiveCurveIRM
			LLTV: big.NewInt(0).SetBytes(common.HexToHash(
				"0x00000000000000000000000000000000000000000000000bee15e785b06c0000").Bytes()),
			// 860000000000000000 = 86%
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
		},

		// ─────────────────────────────────────────────────────────────
		// 2. WETH / USDC — 86% LLTV
		//    Second plus gros marché sur Base par borrow volume
		// ─────────────────────────────────────────────────────────────
		{
			ID: [32]byte(common.HexToHash(
				"0xf10437266b9dd52751bd6255e15cccd0cdf5c75b58c1a3e2621130c905cd8ed9")),
			ChainID:         8453,
			LoanToken:       common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"), // USDC
			CollateralToken: common.HexToAddress("0x4200000000000000000000000000000000000006"), // WETH
			Oracle:          common.HexToAddress("0x4E08B779fD4AB374bCe9D36aE88c3Dbc36dCb48A"),
			IRM:             common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687"), // AdaptiveCurveIRM
			LLTV: big.NewInt(0).SetBytes(common.HexToHash(
				"0x00000000000000000000000000000000000000000000000bee15e785b06c0000").Bytes()),
			// 860000000000000000 = 86%
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
		},

		// ─────────────────────────────────────────────────────────────
		// 3. cbBTC / USDC — 91.5% LLTV (variante plus agressive)
		//    Même pair que #1 mais LLTV plus élevé, marché distinct
		// ─────────────────────────────────────────────────────────────
		{
			ID: [32]byte(common.HexToHash(
				"0x3b3769cfca57be2eaed0f18b9d049fcbeafbe7ca3b6109ebf7d85f22daefe456")),
			ChainID:         8453,
			LoanToken:       common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"), // USDC
			CollateralToken: common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"), // cbBTC
			Oracle:          common.HexToAddress("0x663BECd10daE6C4A3Dcd89F1d76c1174199639B9"),
			IRM:             common.HexToAddress("0x46415998764C29aB2a25CbeA6254146D50D22687"), // AdaptiveCurveIRM
			LLTV: big.NewInt(0).SetBytes(common.HexToHash(
				"0x00000000000000000000000000000000000000000000000caa35e978b9c40000").Bytes()),
			// 915000000000000000 = 91.5%
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
		},
	}

	MorphoBlueAddr = w3.A("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb") // Morpho Blue mainnet

	// ------------------------ FUNC --------------------------------------------------
	// La (pool)
	// market(id) → (totalSupplyAssets, totalSupplyShares, totalBorrowAssets, totalBorrowShares, lastUpdate, fee)
	MarketFunc = w3.MustNewFunc(
		"market(bytes32)",
		"uint128,uint128,uint128,uint128,uint128,uint128",
	)

	// position(id, user) → (supplyShares, borrowShares, collateral)
	PositionFunc = w3.MustNewFunc(
		"position(bytes32,address)",
		"uint256,uint128,uint128",
	)

	// idToMarketParams(id) → (loanToken, collateralToken, oracle, irm, lltv)
	IdToMarketParamsFunc = w3.MustNewFunc(
		"idToMarketParams(bytes32)",
		"address,address,address,address,uint256",
	)

	OraclePriceFunc = w3.MustNewFunc("price()", "uint256")

	// Borrow a 6 params (caller n'est pas indexé mais est bien là)
	EventSupply           = w3.MustNewEvent("Supply(bytes32 indexed id,address,address,uint256,uint256)")
	EventBorrow           = w3.MustNewEvent("Borrow(bytes32 indexed id,address,address,address,uint256,uint256)")
	EventRepay            = w3.MustNewEvent("Repay(bytes32 indexed id,address,address,uint256,uint256)")
	EventLiquidate        = w3.MustNewEvent("Liquidate(bytes32 indexed id,address,address,uint256,uint256,uint256,uint256)")
	EventAccrueInterest   = w3.MustNewEvent("AccrueInterest(bytes32 indexed id,uint256,uint256,uint256)")
	EventSupplyCollateral = w3.MustNewEvent("SupplyCollateral(bytes32 indexed id,address,address,uint256)")
)
