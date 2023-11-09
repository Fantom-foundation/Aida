package utils

type Interval struct {
	first uint64
	last  uint64
	start uint64
	end   uint64
}

func NewInterval(first, last, interval uint64) *Interval {
	f := first - (first % interval)
	return &Interval{first, last, f, f + interval}
}

func (i *Interval) Start() uint64 {
	return Max(i.first, i.start)
}

func (i *Interval) End() uint64 {
	return Min(i.last, i.end)
}

func (i *Interval) Next() *Interval {
	interval = i.end - i.start + 1
	i.start += interval + 1
	i.end += interval
	return i
}
