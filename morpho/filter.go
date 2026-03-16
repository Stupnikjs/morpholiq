package morpho

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

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
