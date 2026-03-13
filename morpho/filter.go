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

// Prend la stuct de reponse api morpho et retourne une liste d'addresse a suivre onchain
type MorphoApiCaller struct {
	Markets []MorphoMarketParams
	// lecture sans lock, zéro contention
}

type BorrowPosition struct {
	MarketID  [32]byte
	Address   common.Address
	HfApi     *big.Int
	HfOnChain *big.Int
}

type ApiPosition struct {
	borrowAssets, borrowAssetsUSD, collateralAssets, collateralAssetsUSD, LLTV *big.Int
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

func (m *MorphoApiCaller) FecthHotPosition(n int) ([]BorrowPosition, error) {
	FilteredPos := []BorrowPosition{}
	for _, m := range m.Markets {
		fetched, err := FecthBorrowersFromMarket(m)
		if err != nil {
			return nil, err
		}

		FilteredPos = append(FilteredPos, fetched...)
		if len(FilteredPos) > n {
			return FilteredPos, err // CHANGER CETTE LIGNE POUR AFFINER LE FILTRE
		}
	}

	return FilteredPos, nil
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

		p := ApiPosition{
			borrowAssets:        ParseBigInt(item.State.BorrowAssets.String()),
			borrowAssetsUSD:     ParseBigInt(item.State.BorrowAssetsUsd.String()),
			collateralAssets:    ParseBigInt(item.State.Collateral.String()),
			collateralAssetsUSD: ParseBigInt(item.State.CollateralAssetsUsd.String()),
			LLTV:                ParseBigInt(item.Market.LLTV.String()),
		}
		hf, in, err := ApplyFilter(p)
		_ = in
		if err != nil {
			return nil, err
		}
		if in {
			FilteredPos = append(FilteredPos, BorrowPosition{
				HfApi:    hf,
				Address:  common.HexToAddress(item.User.Address),
				MarketID: param.ID,
			})

		}
	}
	return FilteredPos, nil
}

func ApplyFilter(p HFparams) (*big.Int, bool, error) {
	hf := HealthFactorUSD(p)
	return hf, true, nil
}
