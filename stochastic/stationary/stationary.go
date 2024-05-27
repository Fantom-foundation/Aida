// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
