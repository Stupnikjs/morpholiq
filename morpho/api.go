package morpho

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/Stupnikjs/morpholiq/utils"
	"github.com/ethereum/go-ethereum/common"
)

type GraphQLRequest struct {
	Query string `json:"query"`
}

type GraphQlResult struct {
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

type Market struct {
	UniqueKey     string `json:"uniqueKey"`
	LLTV          string `json:"lltv"`
	IrmAddress    string `json:"irmAddress"`
	OracleAddress string `json:"oracleAddress"`
	LoanAsset     struct {
		Address  string `json:"address"`
		Symbol   string `json:"symbol"`
		Decimals int    `json:"decimals"`
	} `json:"loanAsset"`
	CollateralAsset struct {
		Address  string `json:"address"`
		Symbol   string `json:"symbol"`
		Decimals int    `json:"decimals"`
	} `json:"collateralAsset"`
}

type Response struct {
	Data struct {
		Markets struct {
			Items []Market `json:"items"`
		} `json:"markets"`
	} `json:"data"`
}

func ApiRespToBorrowPos(params MorphoMarketParams, result GraphQlResult, n int) []BorrowPosition {
	items := result.Data.MarketPositions.Items
	positions := []BorrowPosition{}
	for _, i := range items {
		if i.State.BorrowShares == "0" || i.State.BorrowShares == "" {
			continue
		}
		p := BorrowPosition{
			Address:          common.HexToAddress(i.User.Address),
			BorrowShares:     utils.ParseBigInt(i.State.BorrowShares.String()),
			CollateralAssets: utils.ParseBigInt(i.State.Collateral.String()),
		}
		collUSDScaled := new(big.Int).Mul(utils.ParseBigInt(i.State.CollateralAssetsUsd.String()), utils.TenPowInt(36))
		oraclePrice := new(big.Int).Div(collUSDScaled, utils.ParseBigInt(i.State.CollateralAssetsUsd.String()))
		_ = oraclePrice
		// calculate HF to filter here

		/*
			if hf.Cmp(utils.TenPowInt(6)) < 0 {
				continue // bad debt
			}
			if hf.Cmp(utils.TenPowInt(7)) > 0 {
				continue // bad debt
			}
		*/
		positions = append(positions, p)
	}
	return positions
}

func FecthBorrowersFromMarket(param MorphoMarketParams, n int) ([]BorrowPosition, error) {
	marketIDstr := "0x" + hex.EncodeToString(param.ID[:])

	query := fmt.Sprintf(`{
        "query": "{ marketPositions(first: 10000, where: { marketUniqueKey_in: [\"%s\"], chainId_in: [%d] }) 
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

	var result GraphQlResult

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return ApiRespToBorrowPos(param, result, n), nil

}

func FetchMarkets() ([]Market, error) {
	query := `{
        markets(
            first: 100
            orderBy: SupplyAssetsUsd
            orderDirection: Desc
            where: { chainId_in: [8453] }
        ) {
            items {
                uniqueKey
                lltv
                irmAddress
                oracleAddress
                loanAsset { address symbol decimals }
                collateralAsset { address symbol decimals }
            }
        }
    }`

	body, _ := json.Marshal(GraphQLRequest{Query: query})
	resp, err := http.Post(
		"https://api.morpho.org/graphql",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(data, &result)
	return result.Data.Markets.Items, nil
}

// iterer sur les LLTV pour les plus grosse liquidation
