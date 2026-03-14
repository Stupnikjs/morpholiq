package morpho


// oracle watch list 
// listen to event borrowers 


type PositionsByMarket struct {
      Markets map[[32]byte]*MarketPosition

}
type MarketPosition struct {
   positions map[string]*BorrowPosition 

}

func (mp *MarketPosition) UpdateMarketWithPrice(NewOraclePrice) {
   for _,p := range mp.position {
       p.UpdatePos(NewOraclePrice) 
}

}

func UpdatePosition(newPos){


}