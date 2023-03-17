package exponential

import (
	"fmt"
	"math"
	"math/rand"
)

// Package for the one-sided truncated exponential distribution with a bound of one.

const (
	estimationEps   = 1e-9   // epsilon for bi-section,  and stationary distribution
	approxMaxSteps  = 10000  // maximum number of iterations for finding minimal LSE
	approxInfLambda = 1.0    // lower bound for searching minimal LSE
	approxSupLambda = 1000.0 // upper bound for searching minimal LSE
	dLseEps         = 1e-6   // epsilon for numerical differentiation of the LSE function
)

// Cdf is the cumulative distribution function for the truncated exponential distribution with a bound of 1.
func Cdf(lambda float64, x float64) float64 {
	return (math.Exp(-lambda*x) - 1.0) / (math.Exp(-lambda) - 1.0)
}

// PiecewiseLinearCdf is an approximation of the cumulative distribution function via sampling with n points.
func PiecewiseLinearCdf(lambda float64, n int) [][2]float64 {
	// The points are equi-distantly spread, i.e., 1/n.
	fn := [][2]float64{}
	for i := 0; i <= n; i++ {
		x := float64(i) / float64(n)
		p := Cdf(lambda, x)
		fn = append(fn, [2]float64{x, p})
	}
	return fn
}

// Quantile is the inverse cumulative distribution function for
// producing random values following the exponential distribution
// with parameter lambda (providing probability p).
func Quantile(lambda float64, p float64) float64 {
	return math.Log(p*math.Exp(-lambda)-p+1) / -lambda
}

// DiscreteSample sample the distribution and discretizes the result for numbers between 0 and n-1.
func DiscreteSample(rg *rand.Rand, lambda float64, n int64) int64 {
	return int64(float64(n) * Quantile(lambda, rg.Float64()))
}

// lse is the least square error function for deducing lambda.
func lse(lambda float64, points [][2]float64) float64 {
	err := float64(0.0)
	for i := 0; i < len(points); i++ {
		x := points[i][0]
		p := points[i][1]
		err = err + math.Pow(Cdf(lambda, x)-p, 2)
	}
	return err
}

// dLSE computes the derivative of the least square error function.
// TODO: replace with a symbolic differentiation
func dLSE(lambda float64, points [][2]float64) float64 {
	errL := lse(lambda-dLseEps, points)
	errR := lse(lambda+dLseEps, points)
	return (errR - errL) / dLseEps
}

// ApproximateLambda performs a bisection algorithm to find the best fitting lambda.
func ApproximateLambda(points [][2]float64) (float64, error) {
	// Assumption is that sign of the tangents is in opposite
	// direction for the left and right values of lambda.
	// When left/right values are sufficiently close, the bisection terminates.
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
	return 0.0, fmt.Errorf("ApproximateLambda: failed to converge after %v steps", approxMaxSteps)
}
