package scientist

import (
	"context"
)

const (
	controlBehavior   = "control"
	candidateBehavior = "candidate"
)

func Run[T any](ctx context.Context, name string, setup func(*Experiment[T]) error) (T, error) {
	experiment := New[T](name)
	if err := setup(experiment); err != nil {
		return *new(T), err
	}

	return experiment.Run(ctx)
}
