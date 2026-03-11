package morpho

import (
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

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

type BorrowerStats struct {
	Shares           *big.Int // borrow shares
	BorrowAssets     *big.Int // valeur réelle empruntée
	CollateralAssets *big.Int // collateral déposé
	LLTV             *big.Int // mettre ailleur peut etre

}

type MarketState struct {
	MarketParams       MorphoMarketParams
	BorrowAssetUsd     *big.Float
	CollateralAssetUsd *big.Float
	BorrowerCache      BorrowerCache
}
type BorrowerCache map[common.Address]BorrowerStats

type MorphoEngine struct {
	// lecture sans lock, zéro contention
	snapshot atomic.Pointer[map[[32]byte]MarketState]
}

type BorrowPosition struct {
	MarketID [32]byte
	Address  common.Address
}

type MorphoMarketParams struct {
	ID                      [32]byte
	ChainID                 uint32
	LoanToken               common.Address
	CollateralToken         common.Address
	Oracle                  common.Address
	IRM                     common.Address
	LLTV                    *big.Int // liquidation LTV in WAD (1e18 = 100%)
	LoanTokenDecimals       uint16
	CollateralTokenDecimals uint16
}

type HFManager struct {
	HFMap map[BorrowPosition]*big.Int
}

// scaled by 10e6
type HFparams struct {
	borrowAssets, collateralAssets               *big.Int
	borrowAssetsUSD, collateralAssetsUSD         *big.Float
	borrowAssetDecimals, collateralAssetDecimals uint16
}
