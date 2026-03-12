package morpho

import (
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

type BorrowerStats struct {
	Shares              *big.Int // borrow shares
	BorrowAssets        *big.Int // valeur réelle empruntée
	BorrowAssetsUSD     *big.Int
	CollateralAssets    *big.Int // collateral déposé
	CollateralAssetsUSD *big.Int
	LLTV                *big.Int // mettre ailleur peut etre

}

type MarketState struct {
	MarketParams  MorphoMarketParams
	BorrowerCache BorrowerCache
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

type LiquidablePosition struct {
	BorrowPosition
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	BorrowAssets    *big.Int
	CollateralAsset *big.Int
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
	MarketMap MarketMap
	LLTVmap   map[[32]byte]*big.Int
	HFMap     atomic.Pointer[map[BorrowPosition]*big.Int]
}

// scaled by 10e6
type HFparams struct {
	borrowAssets, collateralAssets               *big.Int
	borrowAssetsUSD, collateralAssetsUSD         *big.Int
	borrowAssetDecimals, collateralAssetDecimals uint16
}

type MarketMap map[[32]byte]MorphoMarketParams
