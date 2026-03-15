package morpho

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (c *PositionCache) AccrueInterestEventProcess(log *types.Log) {
	var (
		id             [32]byte
		prevBorrowRate big.Int
		interest       big.Int
		feeShares      big.Int
	)
	err := EventAccrueInterest.DecodeArgs(log, &id, &prevBorrowRate, &interest, &feeShares)
	if err != nil {
		fmt.Println("decode error:", err)

	}

	// vérifie si ce market est dans ton cache

	if !c.IsMarketInCache(id) {
		return
	}
	fmt.Println("Market in cache")
}

func (c *PositionCache) BorrowEventProcess(log *types.Log) {}

func (c *PositionCache) LiquidateEventProcess(log *types.Log) {
	var (
		id            [32]byte
		borrower      common.Address
		repaidAssets  big.Int
		repaidShares  big.Int
		seizedAssets  big.Int
		badDebtAssets big.Int
		badDebtShares big.Int
	)
	if err := EventLiquidate.DecodeArgs(log, &id, &borrower, &repaidAssets, &repaidShares, &seizedAssets, &badDebtAssets, &badDebtShares); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	if !c.IsMarketInCache(id) {
		return
	}
	market := c.m[id]
	market.Mu.Lock()
	if pos, ok := market.C[borrower]; ok {
		pos.BorrowAssets.Sub(pos.BorrowAssets, &repaidAssets)
		pos.CollateralAssets.Sub(pos.CollateralAssets, &seizedAssets)
		if pos.BorrowAssets.Sign() <= 0 {
			fmt.Printf("Address %s got liquidated \n", pos.Address)
			delete(market.C, borrower)
		}
	}
	market.Mu.Unlock()
}

func (c *PositionCache) RepayEventProcess(log *types.Log) {
	var (
		id       [32]byte
		onBehalf common.Address
		assets   big.Int
		shares   big.Int
	)
	if err := EventRepay.DecodeArgs(log, &id, &onBehalf, &assets, &shares); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	if !c.IsMarketInCache(id) {
		return
	}
	market := c.m[id]
	market.Mu.Lock()
	if pos, ok := market.C[onBehalf]; ok {
		pos.BorrowAssets.Sub(pos.BorrowAssets, &assets)
		if pos.BorrowAssets.Sign() <= 0 {
			fmt.Printf("Address %s repayed debt \n", pos.Address)
			delete(market.C, onBehalf)

		}
	}
	market.Mu.Unlock()
}
