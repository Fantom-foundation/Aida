package stochastic

// IndirectAccess data structure for random access indices permitting deletion without reuse.
type IndirectAccess struct {
	randAcc *RandomAccess

	// translation table for converting compact index space to sparse
	// permitting index deletion without later reuse.
	translation []int

	// counter for introducing new index values
	ctr int
}

// NewIndirectAccess creates a new indirect index access-generator.
func NewIndirectAccess(ra *RandomAccess) *IndirectAccess {
	t := make([]int, ra.numElem)
	for i := 0; i < ra.numElem; i++ {
		t[i] = i + 1
	}
	return &IndirectAccess{
		randAcc:     ra,
		ctr:         ra.numElem,
		translation: t,
	}
}

// NextIndex returns the next index value based on the provided class.
func (a *IndirectAccess) NextIndex(class int) int {
	v := a.randAcc.NextIndex(class)
	if v == -1 {
		return -1
	} else if class == zeroValueID {
		return v
	} else if class == newValueID {
		if v != a.randAcc.numElem {
			panic("unexpected nextIndex result")
		}
		a.ctr++
		v := a.ctr
		a.translation = append(a.translation, v)
		return v
	} else {
		return a.translation[v-1]
	}
}

// findIndex finds the index in the translation table for a given index k.
func (a *IndirectAccess) findIndex(k int) int {
	for i := 0; i < len(a.translation); i++ {
		if a.translation[i] == k {
			return i
		}
	}
	return -1
}

// DeleteIndex deletes an indirect index.
func (a *IndirectAccess) DeleteIndex(k int) error {

	// find index in translation table
	i := a.findIndex(k)
	if i < 0 {
		panic("index not found")
	}

	// delete index i from the translation table and the random access generator.
	a.translation = append(a.translation[:i], a.translation[i+1:]...)
	if err := a.randAcc.DeleteIndex(i); err != nil {
		return err
	}

	return nil
}
