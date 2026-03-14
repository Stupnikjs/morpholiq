package morpho

import (
	"hash/fnv"
	"sync"
)

const numShards = 256


type PositionsByMarket struct {
      Markets map[[32]byte]*MarketPosition

}
type MarketPosition struct {
   positions map[string]*BorrowPosition 

}

func (mp *MarketPosition) UpdateMarket(NewOraclePrice) {
   for _,p := range mp.position {
       p.UpdatePos(NewOraclePrice) 
}

}
