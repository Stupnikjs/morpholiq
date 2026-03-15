package morpho

import (
	"math/big"

	"github.com/Stupnikjs/morpholiq/utils"
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

func (p *BorrowPosition) ApplyFilter(oraclePrice *big.Int) (bool, error) {
	// changer
	p.Hf = p.HealthFactorOraclePrice(oraclePrice)
	return true, nil
}

func (p *BorrowPosition) HealthFactorOraclePrice(oraclePrice *big.Int) *big.Int {
	// HF = coll * oracle / borrowassets * oracle scale
	E36 := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	num := new(big.Int).Mul(p.CollateralAssets, oraclePrice) // collateral * oraclePrice
	num.Mul(num, utils.TenPowInt(6))                         // × 1e6 pour garder la précision
	denom := new(big.Int).Mul(p.BorrowAssets, E36)           // borrowAssets * 1e36
	return new(big.Int).Div(num, denom)
}
