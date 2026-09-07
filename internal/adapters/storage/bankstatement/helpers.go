package bankstatementstorage

import "math"

func roundAmount(v float64) float64 {
	return math.Round(v*10000) / 10000
}
