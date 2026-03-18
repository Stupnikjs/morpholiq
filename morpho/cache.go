package morpho

import (
	"math/big"
	"sync"

	"github.com/Stupnikjs/morpholiq/utils"
	"github.com/ethereum/go-ethereum/common"
)

type Cache struct {
	Markets       []MorphoMarketParams
	OracleCache   *OracleCache
	PositionCache *PositionCache
}

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
	MarketID                       [32]byte
	Address                        common.Address
	BorrowShares, CollateralAssets *big.Int
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

type MorphoMarketParams struct {
	ID              [32]byte
	ChainID         uint32
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	//	IRM                     common.Address
	LLTV                    *big.Int // liquidation LTV in WAD (1e18 = 100%)
	LoanTokenDecimals       uint16
	CollateralTokenDecimals uint16
}

func NewCache(markets []MorphoMarketParams) *Cache {

	return &Cache{
		Markets:       markets,
		OracleCache:   NewOracleCache(markets),
		PositionCache: NewPositionCache(markets),
	}
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
	if totShares.Sign() == 0 {
		return new(big.Int)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(pos.BorrowShares, totBorrowAssets),
		totShares)
}

// prec 1e18
func (pos *BorrowPosition) HF(totShares, totBorrowAssets, oraclePrice, LLTV *big.Int) *big.Int {
	borrowAssets := pos.GetBorrowAssets(totShares, totBorrowAssets)
	if borrowAssets.Sign() == 0 {
		return big.NewInt(0)
	}
	hf := new(big.Int).Div(
		new(big.Int).Mul(pos.CollateralAssets, oraclePrice),
		borrowAssets)

	return new(big.Int).Div(
		new(big.Int).Mul(hf, LLTV),
		utils.TenPowInt(36),
	)
}

func PercentToLiquidation(hf, lltv *big.Int) float64 {
	// distance entre HF et LLTV, en float
	hfF, _ := new(big.Float).SetInt(hf).Float64()
	lltvF, _ := new(big.Float).SetInt(lltv).Float64()
	// les deux sont scaled 1e18, le ratio est donc direct
	return ((hfF - lltvF) / 1e18) * 100
}

// Percent to liquidation
// Évolution of collatéral needed for liquidation

type LiquidablePos struct {
	address     common.Address
	marketIndex int
}

func (c *Cache) LiquidationPotential(threshold float64) []LiquidablePos {
	var liquidable []LiquidablePos

	for i, m := range c.Markets {
		market := c.PositionCache.m[m.ID]

		// oracle d'abord, si absent on skip tout le market
		c.OracleCache.Mu.Lock()
		oracleData := c.OracleCache.C[m.Oracle]
		c.OracleCache.Mu.Unlock()

		if oracleData == nil || oracleData.Price == nil {
			continue
		}

		// lock le market une seule fois pour toute la boucle
		market.Mu.RLock()
		mstats := market.MarketStats

		// garde une copie locale des positions pour libérer le lock vite
		positions := make([]*BorrowPosition, 0, len(market.C))
		for _, p := range market.C {
			positions = append(positions, p)
		}
		market.Mu.RUnlock()

		// calcul sans lock
		for _, p := range positions {
			hf := p.HF(mstats.TotalBorrowShares, mstats.TotalBorrowAssets, oracleData.Price, m.LLTV)
			if hf == nil || hf.Sign() == 0 {
				continue // pas de dette
			}
			// fmt.Println(PercentToLiquidation(hf, m.LLTV))
			if PercentToLiquidation(hf, m.LLTV) < threshold {
				liquidable = append(liquidable, LiquidablePos{
					address:     p.Address,
					marketIndex: i,
				})
			}
		}
	}

	return liquidable
}
