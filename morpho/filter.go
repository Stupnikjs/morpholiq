package morpho

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

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

// Prend la stuct de reponse api morpho et retourne une liste d'addresse a suivre onchain
type MorphoApiCaller struct {
	Markets []MorphoMarketParams
	// lecture sans lock, zéro contention
}

type PositionCache struct {
	m map[[32]byte]MarketCache
}

type MarketCache struct {
	Oracle common.Address
	C      map[common.Address]*BorrowPosition
}

type BorrowPosition struct {
	MarketID                                                                       [32]byte
	Address                                                                        common.Address
	BorrowAssets, BorrowAssetsUSD, CollateralAssets, CollateralAssetsUSD, LLTV, Hf *big.Int
}

func (e *Scanner) RefreshCache(n int) error {

	for _, ma := range e.Markets {
		fetched, err := FecthBorrowersFromMarket(ma)
		if err != nil {
			return err
		}

		for _, p := range fetched {
			e.PositionCache.m[ma.ID].C[p.Address] = &p
		}

	}

	return nil
}

func FecthBorrowersFromMarket(param MorphoMarketParams) ([]BorrowPosition, error) {
	marketIDstr := "0x" + hex.EncodeToString(param.ID[:])

	query := fmt.Sprintf(`{
        "query": "{ marketPositions(first: 1000, where: { marketUniqueKey_in: [\"%s\"], chainId_in: [%d] }) 
		{ items 
		    { user 
			     { address } 
				      state { borrowShares borrowAssets borrowAssetsUsd collateral collateralUsd } 
					   market { lltv }
					  } 
			     } 
		    }"
    }`, marketIDstr, uint(param.ChainID))

	resp, err := http.Post(
		"https://api.morpho.org/graphql",
		"application/json",
		strings.NewReader(query),
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var result struct {
		Data struct {
			MarketPositions struct {
				Items []struct {
					User struct {
						Address string `json:"address"`
					} `json:"user"`
					State struct {
						BorrowShares        json.Number `json:"borrowShares"`
						BorrowAssets        json.Number `json:"borrowAssets"`
						BorrowAssetsUsd     json.Number `json:"borrowAssetsUsd"`
						Collateral          json.Number `json:"collateral"`
						CollateralAssetsUsd json.Number `json:"collateralUsd"`
					} `json:"state"`
					Market struct {
						LLTV json.Number `json:"lltv"`
					}
				} `json:"items"`
			} `json:"marketPositions"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var FilteredPos []BorrowPosition

	for _, item := range result.Data.MarketPositions.Items {
		// Ignorer les positions sans emprunt actif
		if item.State.BorrowShares == "0" || item.State.BorrowShares == "" {
			continue
		}

		p := BorrowPosition{
			BorrowAssets:        ParseBigInt(item.State.BorrowAssets.String()),
			BorrowAssetsUSD:     ParseBigInt(item.State.BorrowAssetsUsd.String()),
			CollateralAssets:    ParseBigInt(item.State.Collateral.String()),
			CollateralAssetsUSD: ParseBigInt(item.State.CollateralAssetsUsd.String()),
			LLTV:                ParseBigInt(item.Market.LLTV.String()),
		}
		p.Address = common.HexToAddress(item.User.Address)
		in, err := p.ApplyFilter(new(big.Int).Div(p.CollateralAssetsUSD, p.BorrowAssetsUSD))
		_ = in
		if err != nil {
			return nil, err
		}
		if in {
			FilteredPos = append(FilteredPos, p)

		}
	}
	return FilteredPos, nil
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
	num.Mul(num, TenPowInt(6))                               // × 1e6 pour garder la précision
	denom := new(big.Int).Mul(p.BorrowAssets, E36)           // borrowAssets * 1e36
	return new(big.Int).Div(num, denom)
}
