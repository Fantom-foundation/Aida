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

package graphutil

import "github.com/onsi/gomega/matchers/support/goraph/bipartitegraph"

// Computes the minimum chain cover of a strict partial order
// using Koenig's bipartite construction and graph matching. The carrier set
// of the strict partial order is represented by ordinal numbers
// from zero to n-1 where n is the cardinality of the carrier
// set. The ordinal numbers are also the topological numbers of the
// strict partial order.

// OrdinalSet represents a subset of the carrier set
type OrdinalSet map[int]struct{}

// StrictPartialOrder stores a strict partial order as a representative function pre: A -> 2^A.
// Iff a ~ b is a relating pair of elements in the strict partial order, then
// element a is in pre(b).  The ordinal numbers coincide with a topological sort
// of the partial order, i.e., for all i: for all j in pre[i]:  i < j.
type StrictPartialOrder []OrdinalSet

// matching is a list of ordinal number pairs that represents matches in the bipartite graph.
// There can be at most n pairs in the matching, and the numbers range from 0 to n-1.
type matching [][2]int

// maxMatching constructs a bipartite graph for the strict partial order using Koenig's construction,
// performs a maximum matching, and returns the matches. See Dilworth's Theorem on Wikipedia for more
// information.
func maxMatching(rel StrictPartialOrder) matching {
	// construct Koenig's bi-partite graph for DAG
	n := len(rel)
	nodes := make([]interface{}, n)
	for i := 0; i < n; i++ {
		nodes[i] = i
	}
	edges := func(x, y interface{}) (bool, error) {
		i := x.(int)
		j := y.(int)
		if _, ok := rel[i][j]; ok {
			return true, nil
		} else {
			return false, nil
		}
	}
	graph, _ := bipartitegraph.NewBipartiteGraph(nodes, nodes, edges)

	// translate matching back to ordinal numbers
	match := matching{}
	for _, edge := range graph.LargestMatching() {
		// use ID (not actual value - naughty!)
		match = append(match, [2]int{edge.Node1, edge.Node2 - n})
	}
	return match
}

// Chain is a list of ordinal numbers which are pairwise-comparable, and the elements are ordered in ascending order.
// The length of the chain is limited by n and the numbers range from 0 to n-1.
type Chain []int

// ChainSet is a set of chains. The number of sets is limited by n.
type ChainSet []Chain

// computeCover constructs the minimum chain cover.
func computeCover(n int, matches matching) ChainSet {

	// initialise minimum chain cover
	minCover := ChainSet{}

	// keep track of processed elements
	processed := OrdinalSet{}

	// iterate over all elements
	for i := 0; i < n; i++ {

		// skip element if it has been processed before
		if _, ok := processed[i]; ok {
			continue
		}

		// found smallest element of chain
		newChain := Chain{i}

		// find remaining chain elements
		j := i
		for {
			foundNext := false
			// TODO: not very efficient because already
			// used edges could be removed from the matching
			// to make the construction linear in the number
			// of edges.
			for _, edge := range matches {
				if edge[1] == j {
					j = edge[0]
					processed[j] = struct{}{}
					newChain = append(newChain, j)
					foundNext = true
				}
			}
			if !foundNext {
				minCover = append(minCover, newChain)
				break
			}
		}
	}

	return minCover
}

// MinChainCover computes the minimum chain cover of a strict partial order.
func MinChainCover(order StrictPartialOrder) ChainSet {
	n := len(order)
	matches := maxMatching(order)
	cover := computeCover(n, matches)
	return cover
}
