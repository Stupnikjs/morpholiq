package morpho

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Borrow Cache + Oracle Cache

type PositionCache struct {
	m map[[32]byte]*MarketCache
}

type OracleCache struct {
	Mu sync.Mutex
	C  map[common.Address]*OracleData
}

type OracleData struct {
	Price *big.Int
	Ts    int64 // Unix timestamp en secondes
}

type MarketCache struct {
	Mu                                   sync.Mutex
	Oracle                               common.Address
	TotalBorrowAssets, totalBorrowShares *big.Int
	C                                    map[common.Address]*BorrowPosition
}

type BorrowPosition struct {
	MarketID                                                                                     [32]byte
	Address                                                                                      common.Address
	BorrowShares, BorrowAssets, BorrowAssetsUSD, CollateralAssets, CollateralAssetsUSD, LLTV, Hf *big.Int
}

func NewOracleCache(params []MorphoMarketParams) *OracleCache {
	return &OracleCache{
		Mu: sync.Mutex{},
		C:  make(map[common.Address]*OracleData, len(params)),
	}
}

func (p *PositionCache) IsMarketInCache(marketID [32]byte) bool {
	market, ok := p.m[marketID]
	return ok && market != nil
}
