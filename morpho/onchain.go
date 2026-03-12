package morpho

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

// reel call a la blockchain pour check la position

func GetPosition(pos BorrowPosition, client *w3.Client, morphoAddress common.Address) {
	var (
		supplyShares      big.Int
		borrowShares      big.Int
		collateral        big.Int
		totalSupplyAssets big.Int
		totalSupplyShares big.Int
		totalBorrowAssets big.Int
		totalBorrowShares big.Int
	)

	err := client.Call(
		eth.CallFunc(morphoAddress, PositionFunc, pos.MarketID, pos.Address).Returns(&supplyShares, &borrowShares, &collateral),
		eth.CallFunc(morphoAddress, MarketFunc, pos.MarketID).Returns(&totalSupplyAssets, &totalSupplyShares, &totalBorrowAssets, &totalBorrowShares, new(big.Int), new(big.Int)),
	)
	if err != nil {
		fmt.Println("err:", err)
		return
	}

	// borrowAssets = borrowShares × totalBorrowAssets / totalBorrowShares
	borrowAssets := new(big.Int).Div(
		new(big.Int).Mul(&borrowShares, &totalBorrowAssets),
		&totalBorrowShares,
	)

	fmt.Printf("Address:       %s\n", pos.Address.Hex())
	fmt.Printf("BorrowShares:  %s\n", borrowShares.String())
	fmt.Printf("BorrowAssets:  %s\n", borrowAssets.String())
	fmt.Printf("BorrowAssets:  %s\n", borrowAssets.String())
	fmt.Printf("Collateral:    %s\n", collateral.String())
}
