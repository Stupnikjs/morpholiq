package morpho

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type GraphQLRequest struct {
	Query string `json:"query"`
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
