package utils

import "math/big"

func ParseBigInt(s string) *big.Int {

	result := new(big.Int)
	if s == "" || s == "0" {
		return result
	}
	result.SetString(s, 10)
	return result

}

func ParseBigFloatToBigInt(s string) *big.Int {
	if s == "" || s == "0" {
		return big.NewInt(0)
	}

	f, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		return big.NewInt(0)
	}

	// scale par 1e18 pour garder les décimales
	scale := new(big.Float).SetInt(TenPowInt(18))
	f.Mul(f, scale)

	result := new(big.Int)
	f.Int(result) // tronque vers zéro
	return result
}

// returns 10 ^ y
func TenPowInt(y uint) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(y)), nil)
}
