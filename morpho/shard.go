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


// subscribe au log 

logsChan := make(chan types.Log, 100)

// Filtre sur le contrat Morpho
filter := ethereum.FilterQuery{
    Addresses: []common.Address{MorphoMain},
}

sub, err := client.Subscribe(
    eth.LogsHandler(filter, func(log types.Log) {
        logsChan <- log
    }),
)
if err != nil {
    log.Fatal(err)
}