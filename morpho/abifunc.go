package morpho

import "github.com/lmittmann/w3"

var (
	MorphoBlueAddr = w3.A("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb") // Morpho Blue mainnet

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
)
