package scientist

type Result[T any] struct {
	Experiment   *Experiment[T]
	Control      *Observation[T, T]
	Observations []*Observation[T, any]
	Candidates   []*Observation[T, any]
	Ignored      []*Observation[T, any]
	Mismatched   []*Observation[T, any]
	Errors       []ResultError
}

func (r Result[T]) IsMatched() bool {
	if r.IsMismatched() || r.IsIgnored() {
		return false
	}
	return true
}

func (r Result[T]) IsMismatched() bool {
	return len(r.Mismatched) > 0
}

func (r Result[T]) IsIgnored() bool {
	return len(r.Ignored) > 0
}

func (r *Result[T]) finalize() {
	if r.Control == nil {
		return
	}

	for _, candidate := range r.Candidates {
		ok, err := r.matching(r.Control, candidate)
		if err != nil {
			ok = false
			r.addError("compare", err)
		}

		if ok {
			continue
		}

		ignored, err := r.ignoring(r.Control, candidate)
		if err != nil {
			ignored = false
			r.addError("ignore", err)
		}

		if ignored {
			r.Ignored = append(r.Ignored, candidate)
			candidate.Ignored = true
		} else {
			r.Mismatched = append(r.Mismatched, candidate)
			candidate.Mismatched = true
		}
	}
}

func (r *Result[T]) addError(operation string, err error) {
	r.Errors = append(r.Errors, ResultError{Operation: operation, Experiment: r.Experiment.Name, Err: err})
}

func (r *Result[T]) matching(control *Observation[T, T], candidate *Observation[T, any]) (bool, error) {
	// neither returned errors
	if control.Err == nil && candidate.Err == nil {
		return r.Experiment.comparator(control.Value, candidate.Value)
	}

	// both returned errors
	if control.Err != nil && candidate.Err != nil {
		return control.Err.Error() == candidate.Err.Error(), nil
	}

	// returned different errors
	return false, nil
}

func (r *Result[T]) ignoring(control *Observation[T, T], candidate *Observation[T, any]) (bool, error) {
	for _, i := range r.Experiment.ignores {
		ok, err := i(control.Value, candidate.Value)
		if err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

type ResultError struct {
	Operation  string
	Experiment string
	Err        error
}

func (e ResultError) Error() string {
	return e.Err.Error()
}
