package morpho

import (
	"time"

	"github.com/lmittmann/w3"
)

type Scanner struct {
	Client    *w3.Client
	ApiCaller *MorphoApiCaller
}

func NewScanner(client *w3.Client, markets []MorphoMarketParams) *Scanner {

	return &Scanner{
		Client:    client,
		ApiCaller: &MorphoApiCaller{Markets: markets},
	}
}

func (e *Scanner) Scan() error {

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			for _, p := range markets {

			}
			// rebuild HFMap après refresh

		}
	}()

	for {
		time.Sleep(1 * time.Second)

		// estimer tout les liquidables
		// onchainHF => EstimateProfit()

	}
}
