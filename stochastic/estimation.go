package stochastic

import (
	"fmt"
	"log"
	"math"

	"gonum.org/v1/gonum/mat"
)

const (
	estimationEps   = 1e-9   // epsilon for bi-section,  and stationary distribution
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
		if math.Abs(right-left) < estimationEps {
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

// StationaryDistribution computes the stationary distribution for a stochastic matrix.
func StationaryDistribution(M [][]float64) ([]float64, error) {
	// flatten matrix for gonum package
	n := len(M)
	elements := []float64{}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			elements = append(elements, M[i][j])
		}
	}
	a := mat.NewDense(n, n, elements)

	// perform eigenvalue decomposition
	var eig mat.Eigen
	ok := eig.Factorize(a, mat.EigenLeft)
	if !ok {
		return nil, fmt.Errorf("eigen-value decomposition failed")
	}

	// find index for eigenvalue of one
	// (note that it is not necessarily the first index)
	v := eig.Values(nil)
	k := -1
	for i, eigenValue := range v {
		if math.Abs(real(eigenValue)-1.0) < estimationEps && math.Abs(imag(eigenValue)) < estimationEps {
			k = i
		}
	}
	if k == -1 {
		return nil, fmt.Errorf("eigen-decomposition failed; no eigenvalue of one found")
	}

	// find left eigenvectors of decomposition
	var ev mat.CDense
	eig.LeftVectorsTo(&ev)

	// compute total for eigenvector with eigenvalue of one.
	total := complex128(0)
	for i := 0; i < n; i++ {
		total += ev.At(i, k)
	}
	if imag(total) > estimationEps {
		return nil, fmt.Errorf("eigen-decomposition failed; eigen-vector is a complex number")
	}

	// normalize eigenvector by total
	stationary := []float64{}
	for i := 0; i < n; i++ {
		stationary = append(stationary, math.Abs(real(ev.At(i, k))/real(total)))
	}
	return stationary, nil
}
