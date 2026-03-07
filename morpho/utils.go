package morpho

import "math/big"

func ParseBigInt(s string) *big.Int {

	result := new(big.Int)
	if s == "" || s == "0" {
		return result
	}
	result.SetString(s, 10)
	return result

}

func ParseBigFloat(s string) *big.Float {

	result := new(big.Float)
	if s == "" || s == "0" {
		return result
	}
	result.SetString(s)
	return result

}
