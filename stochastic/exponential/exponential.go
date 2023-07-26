package exponential

import (
	"fmt"
	"math"
	"math/rand"
)

// Package for the one-sided truncated exponential distribution with a bound of one.

const (
	newtonError      = 1e-9  // epsilon for Newton's convergences criteria
	newtonMaxStep    = 10000 // maximum number of iteration in the Newtonian
	newtonInitLambda = 1.0   // initial parameter in Newtonion's search
)

// Cdf is the cumulative distribution function for the truncated exponential distribution with a bound of 1.
func Cdf(lambda float64, x float64) float64 {
	return (math.Exp(-lambda*x) - 1.0) / (math.Exp(-lambda) - 1.0)
}

// PiecewiseLinearCdf is a piecewise linear representation of the cumulative distribution function.
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

// Quantile is the inverse cumulative distribution function.
func Quantile(lambda float64, p float64) float64 {
	return math.Log(p*math.Exp(-lambda)-p+1) / -lambda
}

// DiscreteSample samples the distribution and discretizes the result for numbers in the range between 0 and n-1.
func DiscreteSample(rg *rand.Rand, lambda float64, n int64) int64 {
	return int64(float64(n) * Quantile(lambda, rg.Float64()))
}

// mean calculates the mean of the empirical cumulative distribution function.
func mean(points [][2]float64) float64 {
	m := float64(0.0)
	for i := 1; i < len(points); i++ {
		x1 := points[i-1][0]
		y1 := points[i-1][1]
		x2 := points[i][0]
		y2 := points[i][1]
		m = m + (x1+x2)*(y2-y1)/2.0
	}
	return m
}

// mle is the Maximum Likelihood Estimation function for finding a suitable lambda.
func mle(lambda float64, mean float64) float64 {
	if math.IsNaN(lambda) || math.IsNaN(mean) {
		panic("Lambda or mean values are not a number")
	}
	t := 1 / (math.Exp(lambda) - 1)
	// ensure that exponent calculation is stable
	if math.IsNaN(t) {
		// If numerical limits are reached, replace with symbolic limits.
		if lambda >= 1.0 {
			t = 0
		} else {
			// assuming that for very small values of lambda, a NaN is produced.
			t = 1.0
		}
	}
	return 1/lambda - t - mean
}

// dMle computes the derivative of the Maximum Likelihood Estimation function.
func dMle(lambda float64) float64 {
	if math.IsNaN(lambda) {
		panic("Lambda or mean values are not a number")
	}
	t := math.Exp(lambda) / math.Pow(math.Exp(lambda)-1, 2)
	// ensure that exponent calculation is stable
	if math.IsNaN(t) {
		// If numerical limits are reached, replace by symbolic limits.
		t = 1.0
	}
	return t - 1/(lambda*lambda)
}

// ApproximateLambda performs a classical Newtonian to determine
// the lambda value since the MLE function is a transcendental
// functions and no closed form can be found. The function returns either
// lambda if it is in the epsilon environment (newtonError) or
// an error if the maximal number of steps for the convergence criteria
// is exceeded.
func ApproximateLambda(points [][2]float64) (float64, error) {
	m := mean(points)
	l := newtonInitLambda
	for step := 0; step < newtonMaxStep; step++ {
		err := mle(l, m)
		l = l - err/dMle(l)
		if math.Abs(err) < newtonError {
			return l, nil
		}
	}
	return 0.0, fmt.Errorf("ApproximateLambda: failed to converge after %v steps", newtonMaxStep)
}
