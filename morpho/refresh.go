package morpho

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

func (c *Cache) ApiRefreshCache() error {

	for _, ma := range c.Markets {
		fetched, err := FetchBorrowersFromMarket(ma)
		if err != nil {
			return err
		}

		for _, p := range fetched {
			c.PositionCache.m[ma.ID].C[p.Address] = &p
		}

	}
	return nil
}

func (c *Cache) OnChainRefresh(client *w3.Client) error {

	ctx := context.Background()
	var calls []w3types.RPCCaller

	marketMap, marketCalls := c.MarketStatsCalls()
	oraclePrice, oracleCalls := c.OracleCalls()

	calls = append(calls, marketCalls...)
	calls = append(calls, oracleCalls...)

	for _, call := range calls {
		if err := client.CallCtx(ctx, call); err != nil {
			return err
		}
	}
	/* for dRPC limit
	if err := client.CallCtx(ctx, calls...); err != nil {
		return err
	}
	*/
	// update
	c.ApplyMarketStats(marketMap)

	// unpack oracle
	c.ApplyOraclePrices(oraclePrice)
	return nil
}

func (c *Cache) MarketStatsCalls() (map[[32]byte]*MarketStats, []w3types.RPCCaller) {
	var calls []w3types.RPCCaller

	marketStates := make(map[[32]byte]*MarketStats, len(c.Markets))

	for id := range c.PositionCache.m {
		ms := MarketStats{
			TotalBorrowAssets: new(big.Int),
			TotalBorrowShares: new(big.Int),
		}

		marketStates[id] = &ms
		calls = append(calls, eth.CallFunc(MorphoMain, MarketFunc, id).Returns(
			new(big.Int), new(big.Int), // supply on s'en fout
			ms.TotalBorrowAssets, ms.TotalBorrowShares,
			new(big.Int), new(big.Int),
		))
	}

	return marketStates, calls
}

// RETOURNE LES ORACLE CALLS AVEC LES POINTEURS DE RESULT
func (c *Cache) OracleCalls() (map[common.Address]*big.Int, []w3types.RPCCaller) {
	oraclePrices := make(map[common.Address]*big.Int)
	var calls []w3types.RPCCaller

	for _, m := range c.PositionCache.m {
		if _, ok := oraclePrices[m.Oracle]; !ok {
			price := new(big.Int)
			oraclePrices[m.Oracle] = price
			calls = append(calls, eth.CallFunc(m.Oracle, OraclePriceFunc).Returns(price))
		}
	}
	return oraclePrices, calls
}

func (c *Cache) ApplyMarketStats(marketMap map[[32]byte]*MarketStats) {
	for id, m := range c.PositionCache.m {
		ms, ok := marketMap[id]
		if !ok {
			continue
		}
		m.Mu.Lock()
		m.MarketStats.TotalBorrowAssets = ms.TotalBorrowAssets
		m.MarketStats.TotalBorrowShares = ms.TotalBorrowShares
		m.Mu.Unlock()
	}
}

func (c *Cache) ApplyOraclePrices(prices map[common.Address]*big.Int) {
	now := time.Now().Unix()
	c.OracleCache.Mu.Lock()
	defer c.OracleCache.Mu.Unlock()

	for addr, price := range prices {
		c.OracleCache.C[addr] = &OracleData{Price: price, Ts: now}
	}
}
