package stochastic

// IndirectAccess data structure for random access indices permitting deletion without reuse.
type IndirectAccess struct {
	randAcc *RandomAccess

	// translation table for converting compact index space to sparse
	// permitting index deletion without later reuse.
	translation []int64

	// counter for introducing new index values
	ctr int64
}

// NewIndirectAccess creates a new indirect index access-generator.
func NewIndirectAccess(ra *RandomAccess) *IndirectAccess {
	t := make([]int64, ra.numElem)
	for i := int64(0); i < ra.numElem; i++ {
		t[i] = i + 1
	}
	return &IndirectAccess{
		randAcc:     ra,
		ctr:         ra.numElem,
		translation: t,
	}
}

// NextIndex returns the next index value based on the provided class.
func (a *IndirectAccess) NextIndex(class int) int64 {
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

// DeleteIndex deletes an indirect index.
func (a *IndirectAccess) DeleteIndex(k int64) error {

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

// findIndex finds the index in the translation table for a given index k.
func (a *IndirectAccess) findIndex(k int64) int64 {
	for i := int64(0); i < int64(len(a.translation)); i++ {
		if a.translation[i] == k {
			return i
		}
	}
	return -1
}

// NumElem returns the number of elements
func (a *IndirectAccess) NumElem() int64 {
	return a.randAcc.numElem
}
