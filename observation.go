package scientist

import "time"

type Observation[TE any, TVal any] struct {
	Experiment *Experiment[TE]
	Name       string
	Started    time.Time
	Runtime    time.Duration
	Value      TVal
	Err        error
	Mismatched bool
	Ignored    bool
}

func (o *Observation[TE, TVal]) CleanedValue() (interface{}, error) {
	return o.Experiment.cleaner(o.Value)
}
