package main

import "github.com/Stupnikjs/morpholiq/morpho"

func main() {
	// Connexion au noeud Ethereum (remplace par ton RPC)
	scanner := morpho.NewScanner(morpho.DRPC, morpho.DWS)

	scanner.Scan()
}

/*

-- Call a l'api morpho pour initier une struct de suivi des liquidations futures // recall toute les 10min
-- tout les block reactualisation des HF
-- Si position liquidable ==> simulation de transaction / calcul du slipage ==> estimation des gains potentiels
-- envoie de la transaction

*/
