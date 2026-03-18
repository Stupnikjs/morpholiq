package morpho

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Stupnikjs/morpholiq/utils"
	"github.com/ethereum/go-ethereum/common"
)

const (
	morphoGraphQLURL = "https://api.morpho.org/graphql"
	defaultPageLimit = 500
)

// ── HTTP ─────────────────────────────────────────────────────────────────────

func graphqlPost(query string) ([]byte, error) {
	body, err := json.Marshal(GraphQLRequest{Query: query})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(morphoGraphQLURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// ── QUERIES ──────────────────────────────────────────────────────────────────

func positionsQuery(marketID string, chainID uint32, limit, skip int) string {
	return fmt.Sprintf(`{
        marketPositions(
            first: %d
            skip: %d
            where: {
                marketUniqueKey_in: ["%s"]
                chainId_in: [%d]
            }
        ) {
            items {
                user { address }
                state { borrowShares collateral }
            }
            pageInfo { countTotal }
        }
    }`, limit, skip, marketID, chainID)
}

// ── FETCH ────────────────────────────────────────────────────────────────────

func FetchBorrowersFromMarket(param MorphoMarketParams) ([]BorrowPosition, error) {
	marketID := "0x" + hex.EncodeToString(param.ID[:])
	var all []BorrowPosition
	skip := 0

	for {
		data, err := graphqlPost(positionsQuery(marketID, param.ChainID, defaultPageLimit, skip))
		if err != nil {
			return nil, fmt.Errorf("graphql fetch (skip=%d): %w", skip, err)
		}

		var result GraphQLResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("graphql decode (skip=%d): %w", skip, err)
		}

		all = append(all, parsePositions(param, result)...)

		total := result.Data.MarketPositions.PageInfo.CountTotal
		skip += defaultPageLimit
		if skip >= total {
			break
		}
	}

	return all, nil
}

// ── PARSING ──────────────────────────────────────────────────────────────────

func parsePositions(params MorphoMarketParams, result GraphQLResult) []BorrowPosition {
	items := result.Data.MarketPositions.Items
	positions := make([]BorrowPosition, 0, len(items))

	for _, item := range items {
		borrowShares := utils.ParseBigInt(item.State.BorrowShares.String())
		collateral := utils.ParseBigInt(item.State.Collateral.String())

		// ignore les positions fermées
		if borrowShares.Sign() == 0 && collateral.Sign() == 0 {
			continue
		}

		positions = append(positions, BorrowPosition{
			MarketID:         params.ID,
			Address:          common.HexToAddress(item.User.Address),
			BorrowShares:     borrowShares,
			CollateralAssets: collateral,
		})
	}
	return positions
}

// ── TYPES ────────────────────────────────────────────────────────────────────

type GraphQLRequest struct {
	Query string `json:"query"`
}

type GraphQLResult struct {
	Data struct {
		MarketPositions struct {
			Items []struct {
				User struct {
					Address string `json:"address"`
				} `json:"user"`
				State struct {
					BorrowShares json.Number `json:"borrowShares"`
					Collateral   json.Number `json:"collateral"`
				} `json:"state"`
			} `json:"items"`
			PageInfo struct {
				CountTotal int `json:"countTotal"`
			} `json:"pageInfo"`
		} `json:"marketPositions"`
	} `json:"data"`
}
