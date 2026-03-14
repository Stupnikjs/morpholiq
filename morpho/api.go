package morpho

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

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

func ApiRespToBorrowPos(result GraphQlResult) []BorrowPosition {
	items := result.Data.MarketPositions.Items
	positions := []BorrowPosition{}
	for _, i := range items {
		if i.State.BorrowShares == "0" || i.State.BorrowShares == "" {
			continue
		}
		positions = append(positions, BorrowPosition{
			Address:             common.HexToAddress(i.User.Address),
			BorrowAssets:        ParseBigInt(i.State.BorrowAssets.String()),
			BorrowAssetsUSD:     ParseBigInt(i.State.BorrowAssetsUsd.String()),
			CollateralAssets:    ParseBigInt(i.State.Collateral.String()),
			CollateralAssetsUSD: ParseBigInt(i.State.CollateralAssetsUsd.String()),
			LLTV:                ParseBigInt(i.Market.LLTV.String()),
		})
	}
	return positions
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
