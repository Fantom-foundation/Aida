package stochastic

import (
	"fmt"
	"log"
	"math"

	"gonum.org/v1/gonum/mat"
)

const (
	approxEps       = 1e-9   // epsilon for terminating the bi-section for finding minimal LSE
	approxMaxSteps  = 10000  // maximum number of iterations for finding minimal LSE
	approxInfLambda = 1.0    // lower bound for searching minimal LSE
	approxSupLambda = 1000.0 // upper bound for searching minimal LSE

	dLseEps = 1e-6 // epsilon for numerical differentiation of the LSE function
)

// EstimationModelJSON is the output of the estimator in JSON format.
type EstimationModelJSON struct {
	Operations       []string    `json:"operations"`
	StochasticMatrix [][]float64 `json:"stochasticMatrix"`

	Contracts EstimationStatsJSON `json:"contractStats"`
	Keys      EstimationStatsJSON `json:"keyStats"`
	Values    EstimationStatsJSON `json:"valueStats"`
}

// EstimationStatsJSON is an estimated access statistics in JSON format.
type EstimationStatsJSON struct {
	NumKeys           int       `json:"n"`
	Lambda            float64   `json:"exponentialParameter"`
	QueueDistribution []float64 `json:"queuingDistribution"`
}

// NewEstimationModelJSON creates a new estimation model.
func NewEstimationModelJSON(d *EventRegistryJSON) EstimationModelJSON {
	// copy operation codes
	operations := make([]string, len(d.Operations))
	copy(operations, d.Operations)

	// copy stochastic matrix
	stochasticMatrix := make([][]float64, len(d.StochasticMatrix))
	for i := range d.StochasticMatrix {
		stochasticMatrix[i] = make([]float64, len(d.StochasticMatrix[i]))
		copy(stochasticMatrix[i], d.StochasticMatrix[i])
	}

	return EstimationModelJSON{
		Operations:       operations,
		StochasticMatrix: stochasticMatrix,
		Contracts:        NewEstimationStats(&d.Contracts),
		Keys:             NewEstimationStats(&d.Keys),
		Values:           NewEstimationStats(&d.Values),
	}
}

// NewEstimationStats creates a new EstimationStatsJSON objects for an access statistics.
func NewEstimationStats(d *AccessStatsJSON) EstimationStatsJSON {
	// compute lambda
	lambda, err := ApproximateLambda(d.CountingStats.ECdf)
	if err != nil {
		log.Fatalf("Failed to approximate lambda parameter. Error: %v", err)
	}

	// copy queuing distribution
	distribution := make([]float64, len(d.QueuingStats.Distribution))
	copy(distribution, d.QueuingStats.Distribution)

	return EstimationStatsJSON{
		Lambda:            lambda,
		NumKeys:           d.CountingStats.NumKeys,
		QueueDistribution: distribution,
	}
}

// Cdf is the cumulative distribution function for the truncated exponential distribution with a bound of 1.
func Cdf(lambda float64, x float64) float64 {
	return (math.Exp(-lambda*x) - 1.0) / (math.Exp(-lambda) - 1.0)
}

// LSE is the least square error function between Cdf and eCDF with parameter lambda.
func LSE(lambda float64, points [][2]float64) float64 {
	err := float64(0.0)
	for i := 0; i < len(points); i++ {
		x := points[i][0]
		p := points[i][1]
		err = err + math.Pow(Cdf(lambda, x)-p, 2)
	}
	return err
}

// dLSE computes the derivative of the least square error function.
func dLSE(lambda float64, points [][2]float64) float64 {
	errL := LSE(lambda-dLseEps, points)
	errR := LSE(lambda+dLseEps, points)
	return (errR - errL) / dLseEps
}

// ApproximateLambda applies a bisection algorithm to find the best fitting lambda by minimising the LSE.
func ApproximateLambda(points [][2]float64) (float64, error) {
	left := approxInfLambda
	right := approxSupLambda
	for i := 0; i < approxMaxSteps; i++ {
		mid := (right + left) / 2.0
		dErr := dLSE(mid, points)
		// check direction of LSE's tangent
		if dErr > 0.0 {
			right = mid
		} else {
			left = mid
		}
		if math.Abs(right-left) < approxEps {
			return mid, nil
		}
	}
	return 0.0, fmt.Errorf("Failed to converge after %v steps", approxMaxSteps)
}

// PiecewiseLinearCdf is an approximation of the cumulative distribution function via sampling with n points.
func PiecewiseLinearCdf(lambda float64, n int) [][2]float64 {
	// The points are equi-distantly spread, i.e., 1/n.
	fn := [][2]float64{}
	for i := 0; i <= numDistributionPoints; i++ {
		x := float64(i) / float64(n)
		p := Cdf(lambda, x)
		fn = append(fn, [2]float64{x, p})
	}
	return fn
}

// SteadyStateDistribution computes the steady-state for a stochastic matrix.
func SteadyStateDistribution(M [][]float64) []float64 {
	// compute eigen-value/vector of stochastic matrix
	n := len(M)
	elements := []float64{}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			elements = append(elements, M[i][j])
		}
	}
	a := mat.NewDense(n, n, elements)
	var eig mat.Eigen
	ok := eig.Factorize(a, mat.EigenLeft)
	if !ok {
		log.Fatal("eigen-decomposition failed")
	}
	var ev mat.CDense
	eig.LeftVectorsTo(&ev)
	// compute total of the first left eigenvector
	total := float64(0)
	for i := 0; i < n; i++ {
		total += math.Abs(real(ev.At(i, 0)))
	}
	// steady state is the normalized, first left eigenvector
	steadyState := []float64{}
	for i := 0; i < n; i++ {
		steadyState = append(steadyState, math.Abs(real(ev.At(i, 0)))/total)
	}
	return steadyState
}
