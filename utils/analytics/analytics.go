// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package analytics

// Interface for any further analytics implementation
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
