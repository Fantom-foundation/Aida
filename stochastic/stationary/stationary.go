package stationary

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

const (
	estimationEps = 1e-9 // epsilon for stationary distribution
)

// ComputeDistribution computes the stationary distribution of a stochastic matrix.
func ComputeDistribution(M [][]float64) ([]float64, error) {
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
