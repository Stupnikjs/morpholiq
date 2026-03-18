package morpho

import (
	"math/big"

	"github.com/Stupnikjs/morpholiq/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

var (
	DRPC         = "https://lb.drpc.live/ethereum/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	DWS          = "wss://lb.drpc.live/ethereum/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	INFURAMAIN   = "https://mainnet.infura.io/v3/e587127983764e6284261ebf6b4aaedf"
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
			LLTV: big.NewInt(0).SetBytes(common.HexToHash(
				"0x00000000000000000000000000000000000000000000000bee15e785b06c0000").Bytes()),
			// 860000000000000000 = 86%

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

			LLTV: big.NewInt(0).SetBytes(common.HexToHash(
				"0x00000000000000000000000000000000000000000000000bee15e785b06c0000").Bytes()),
			// 860000000000000000 = 86%
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

			LLTV: big.NewInt(0).SetBytes(common.HexToHash(
				"0x00000000000000000000000000000000000000000000000caa35e978b9c40000").Bytes()),
			// 915000000000000000 = 91.5%

		},
	}

	MainnetParams = []MorphoMarketParams{
		// wstETH / WETH — 96.5% LLTV
		{
			ID:                      [32]byte(common.HexToHash("0xb8fc70e82bc5bb53e773626fcc6a23f7eefa036918d7ef216ecfb1950a94a85e")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), // WETH
			CollateralToken:         common.HexToAddress("0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"), // wstETH
			Oracle:                  common.HexToAddress("0xbD60A6770b27E084E8617335ddE769241B0e71D8"),
			LLTV:                    utils.ParseBigInt("965000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
		},
		// cbBTC / USDC — 86% LLTV
		{
			ID:                      [32]byte(common.HexToHash("0x64d65c9a2d91c36d56fbc42d69e979335320169b3df63bf92789e2c8883fcc64")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // USDC
			CollateralToken:         common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"), // cbBTC
			Oracle:                  common.HexToAddress("0xA6D6950c9F177F1De7f7757FB33539e3Ec60182a"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
		},
		// WBTC / USDC — 86% LLTV
		{
			ID:                      [32]byte(common.HexToHash("0x3a85e619751152991742810df6ec69ce473daef99e28a64ab2340d7b7ccfee49")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // USDC
			CollateralToken:         common.HexToAddress("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"), // WBTC
			Oracle:                  common.HexToAddress("0xDddd770BADd886dF3864029e4B377B5F6a2B6b83"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
		},
		// WBTC / USDT — 86% LLTV
		{
			ID:                      [32]byte(common.HexToHash("0xa921ef34e2fc7a27ccc50ae7e4b154e16c9799d3387076c421423ef52ac4df99")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
			CollateralToken:         common.HexToAddress("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"), // WBTC
			Oracle:                  common.HexToAddress("0x008bF4B1cDA0cc9f0e882E0697f036667652E1ef"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 8,
		},
		// wstETH / USDT — 86% LLTV
		{
			ID:                      [32]byte(common.HexToHash("0xe7e9694b754c4d4f7e21faf7223f6fa71abaeb10296a4c43a54a7977149687d2")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
			CollateralToken:         common.HexToAddress("0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"), // wstETH
			Oracle:                  common.HexToAddress("0x95DB30fAb9A3754e42423000DF27732CB2396992"),
			LLTV:                    utils.ParseBigInt("860000000000000000"),
			LoanTokenDecimals:       6,
			CollateralTokenDecimals: 18,
		},
		// weETH / WETH — 94.5% LLTV
		{
			ID:                      [32]byte(common.HexToHash("0x37e7484d642d90f14451f1910ba4b7b8e4c3ccdd0ec28f8b2bdb35479e472ba7")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), // WETH
			CollateralToken:         common.HexToAddress("0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"), // weETH
			Oracle:                  common.HexToAddress("0xbDd2F2D473E8D63d1BFb0185B5bDB8046ca48a72"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       18,
			CollateralTokenDecimals: 18,
		},
		// LBTC / WBTC — 94.5% LLTV
		{
			ID:                      [32]byte(common.HexToHash("0xf6a056627a51e511ec7f48332421432ea6971fc148d8f3c451e14ea108026549")),
			ChainID:                 1,
			LoanToken:               common.HexToAddress("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"), // WBTC
			CollateralToken:         common.HexToAddress("0x8236a87084f8B84306f72007F36F2618A5634494"), // LBTC
			Oracle:                  common.HexToAddress("0xa98105B8227E0f2157816Feb7A331364A9B74F80"),
			LLTV:                    utils.ParseBigInt("945000000000000000"),
			LoanTokenDecimals:       8,
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
