package morpho

import (
	"math/big"
	"sync"

	"github.com/Stupnikjs/morpholiq/utils"
	"github.com/ethereum/go-ethereum/common"
)

type PositionCache struct {
	m map[[32]byte]*Market
}

type Market struct {
	Mu sync.RWMutex
	MarketCache
	MarketStats
}

type MarketStats struct {
	TotalBorrowAssets, TotalBorrowShares, LLTV *big.Int
}

type MarketCache struct {
	Oracle common.Address
	C      map[common.Address]*BorrowPosition
}

type BorrowPosition struct {
	MarketID                           [32]byte
	Address                            common.Address
	BorrowShares, CollateralAssets, Hf *big.Int
}

/*
   ______________________________________________________________________________________

*/

type OracleCache struct {
	Mu sync.Mutex
	C  map[common.Address]*OracleData
}

type OracleData struct {
	Price *big.Int
	Ts    int64 // Unix timestamp en secondes
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

func (pos *BorrowPosition) GetBorrowAssets(totShares, totBorrowAssets *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Mul(pos.BorrowShares, totBorrowAssets),
		totShares)
}

// prec 1e18
func (pos *BorrowPosition) HF(totShares, totBorrowAssets, oraclePrice, LLTV *big.Int) *big.Int {
	borrowAssets := pos.GetBorrowAssets(totShares, totBorrowAssets)
	hf := new(big.Int).Div(
		new(big.Int).Mul(pos.CollateralAssets, oraclePrice),
		borrowAssets)

	return new(big.Int).Div(
		new(big.Int).Mul(hf, LLTV),
		utils.TenPowInt(36),
	)
}


// Percent to liquidation 
// Évolution of collatéral needed for liquidation