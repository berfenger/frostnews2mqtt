package sunspec

import (
	"math"
)

func applySF(number uint16, sf uint16) float64 {
	return float64(number) * math.Pow(10, float64(int16(sf)))
}

func applySFInv(number uint16, sf uint16) float64 {
	return float64(number) / math.Pow(10, float64(int16(sf)))
}

func applySFint16(number int16, sf uint16) float64 {
	return float64(number) * math.Pow(10, float64(int16(sf)))
}

func applySFuint32(number uint32, sf uint16) float64 {
	return float64(number) * math.Pow(10, float64(int16(sf)))
}

func applySFfloat64Inv(number float64, sf uint16) float64 {
	return number / math.Pow(10, float64(int16(sf)))
}
