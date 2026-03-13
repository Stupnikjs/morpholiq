package morpho

import (
	"fmt"
	"time"

	"github.com/lmittmann/w3"
)

type Scanner struct {
	Client    *w3.Client
	ApiCaller *MorphoApiCaller
	WatchList PositionStore
}

func NewScanner(client *w3.Client, markets []MorphoMarketParams) *Scanner {

	return &Scanner{
		Client:    client,
		ApiCaller: &MorphoApiCaller{Markets: markets},
		WatchList: *NewPositionStore(),
	}
}

func (e *Scanner) Refresh() error {
	bp, err := e.ApiCaller.FecthHotPosition(10)
	fmt.Println(len(bp))
	if err != nil {
		return err
	}
	for _, p := range bp {
		e.WatchList.Set(p.Address.String(), &p)
	}
	return nil

}

func (e *Scanner) Scan() error {

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			e.Refresh()
		}
	}()

	for {
		time.Sleep(1 * time.Second)
		// estimer tout les liquidables
		// onchainHF => EstimateProfit()

	}
}
