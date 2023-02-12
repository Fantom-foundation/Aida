package stochastic

// IndirectAccess data structure for producing random accesses.
type IndirectAccess struct {
	randAcc *RandomAccess

	// queue for indexes
	translation []int

	// counter
	ctr int
}

// NewAccessStats creates a new access index.
func NewIndirectAccess(ra *RandomAccess) *IndirectAccess {
	t := make([]int, ra.numElem)
	for i := 0; i < ra.numElem; i++ {
		t[i] = i + 1
	}
	// initialise table!
	return &IndirectAccess{
		randAcc:     ra,
		ctr:         ra.numElem,
		translation: t,
	}
}

// NextIndex returns the next random index based on the provided class.
func (a *IndirectAccess) NextIndex(class int) int {
	switch class {
	case zeroValueID:
		return 0
	case newValueID:
		a.ctr++
		v := a.ctr
		a.translation = append(a.translation, v)
		return v
	case previousValueID:
		return a.translation[a.NextIndex(class)-1]
	case recentValueID:
		return a.translation[a.NextIndex(class)-1]
	case randomValueID:
		return a.translation[a.NextIndex(class)-1]
	default:
		return -1
	}
}

// DeleteIndex deletes an access index.
func (a *IndirectAccess) DeleteIndex(i int) error {

	// delete element (by changing order)
	a.translation[i] = a.translation[len(a.translation)-1]
	a.translation[len(a.translation)-1] = 0
	a.translation = a.translation[:len(a.translation)-1]

	// delete element from queue and reduce cardinality
	err := a.DeleteIndex(i)
	if err != nil {
		return err
	}

	return nil
}
