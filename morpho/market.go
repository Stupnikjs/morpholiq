package morpho

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
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

type MorphoMarketParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	IRM             common.Address
	LLTV            *big.Int // liquidation LTV in WAD (1e18 = 100%)
}

type MorphoMarketState struct {
	ID                  [32]byte
	Params              MorphoMarketParams
	TotalSupplyAssets   *big.Int
	TotalSupplyShares   *big.Int
	TotalBorrowAssets   *big.Int
	TotalBorrowShares   *big.Int
	LastUpdate          *big.Int
	Fee                 *big.Int
	UtilizationRate     float64 // TotalBorrowAssets / TotalSupplyAssets  [0-1]
	LiquidationPressure float64 // how close borrow is to supply ceiling    [0-1]
}

// MorphoPosition holds a user's position inside a specific market.
type MorphoPosition struct {
	MarketID     [32]byte
	User         common.Address
	SupplyShares *big.Int
	BorrowShares *big.Int
	Collateral   *big.Int
}

func GetMorphoMarketState(client *w3.Client, marketID [32]byte) (*MorphoMarketState, error) {
	ctx := context.Background()

	// --- decode market(id) returns 6 uint128 packed values ---
	var (
		totalSupplyAssets, totalSupplyShares *big.Int
		totalBorrowAssets, totalBorrowShares *big.Int
		lastUpdate, fee                      *big.Int
	)

	idBytes := marketID // bytes32 arg

	if err := client.CallCtx(ctx,
		eth.CallFunc(MorphoBlueAddr, MarketFunc, idBytes).
			Returns(&totalSupplyAssets, &totalSupplyShares,
				&totalBorrowAssets, &totalBorrowShares,
				&lastUpdate, &fee),
	); err != nil {
		return nil, fmt.Errorf("morpho market(%x): %w", marketID, err)
	}

	// --- decode idToMarketParams(id) ---
	var (
		loanToken, collateralToken, oracle, irm common.Address
		lltv                                    *big.Int
	)
	if err := client.CallCtx(ctx,
		eth.CallFunc(MorphoBlueAddr, IdToMarketParamsFunc, idBytes).
			Returns(&loanToken, &collateralToken, &oracle, &irm, &lltv),
	); err != nil {
		return nil, fmt.Errorf("morpho idToMarketParams(%x): %w", marketID, err)
	}

	// --- compute utilisation rate ---
	var utilRate float64
	if totalSupplyAssets != nil && totalSupplyAssets.Sign() > 0 {
		supply := new(big.Float).SetInt(totalSupplyAssets)
		borrow := new(big.Float).SetInt(totalBorrowAssets)
		r, _ := new(big.Float).Quo(borrow, supply).Float64()
		utilRate = r
	}

	// --- liquidation pressure: how close utilisation is to LLTV ceiling ---
	// LLTV is in WAD (1e18 == 100%), convert to [0-1] float first.
	var liqPressure float64
	if lltv != nil && lltv.Sign() > 0 {
		wad := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
		lltvF, _ := new(big.Float).Quo(new(big.Float).SetInt(lltv), wad).Float64()
		if lltvF > 0 {
			liqPressure = utilRate / lltvF // > 1.0 means technically insolvent pool
		}
	}

	return &MorphoMarketState{
		ID: marketID,
		Params: MorphoMarketParams{
			LoanToken:       loanToken,
			CollateralToken: collateralToken,
			Oracle:          oracle,
			IRM:             irm,
			LLTV:            lltv,
		},
		TotalSupplyAssets:   totalSupplyAssets,
		TotalSupplyShares:   totalSupplyShares,
		TotalBorrowAssets:   totalBorrowAssets,
		TotalBorrowShares:   totalBorrowShares,
		LastUpdate:          lastUpdate,
		Fee:                 fee,
		UtilizationRate:     utilRate,
		LiquidationPressure: liqPressure,
	}, nil
}
