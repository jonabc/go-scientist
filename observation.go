package scientist

import "time"

type Observation[TE any, TVal any] struct {
	Experiment *Experiment[TE]
	Name       string
	Started    time.Time
	Runtime    time.Duration
	Value      TVal
	Err        error
}

func (o *Observation[TE, TVal]) CleanedValue() (interface{}, error) {
	return o.Experiment.cleaner(o.Value)
}
