package profile

type Analytics interface {
	Reset()
	Update(id byte, data float64)

	GetCount(id byte) uint64
	GetMin(id byte) float64
	GetMax(id byte) float64

	GetSum(id byte) float64
	GetMean(id byte) float64
	GetStandardDeviation(id byte) float64
	GetVariance(id byte) float64
	GetSkewness(id byte) float64
	GetKurtosis(id byte) float64
}
