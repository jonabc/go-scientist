package scientist

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestExperimentMatch(t *testing.T) {
	e := New[int]("match")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (interface{}, error) {
		return 1, nil
	})

	published := false
	e.Publish(func(r *Result[int]) error {
		published = true

		if !r.IsMatched() || r.IsMismatched() {
			t.Errorf("not matched")
		}

		if r.IsIgnored() {
			t.Errorf("ignored")
		}

		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !published {
		t.Errorf("expected Publish callback to run")
	}
}

func TestExperimentMismatchNoReturn(t *testing.T) {
	e := New[int]("match")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 2, nil
	})

	published := false
	e.Publish(func(r *Result[int]) error {
		published = true

		if r.IsMatched() || !r.IsMismatched() {
			t.Errorf("matched???")
		}

		if r.IsIgnored() {
			t.Errorf("ignored")
		}

		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !published {
		t.Errorf("expected Publish callback to run")
	}
}

func TestExperimentRunBefore(t *testing.T) {
	runIf := false
	before := false

	e := New[int]("run")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 1, nil
	})

	e.RunIf(func() (bool, error) {
		runIf = true
		return true, nil
	})

	e.BeforeRun(func() error {
		before = true
		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !runIf {
		t.Errorf("expected RunIf callback to run")
	}

	if !before {
		t.Errorf("expected BeforeRun callback to run")
	}
}

func TestExperimentDisabledRunBefore(t *testing.T) {
	runIf := false

	e := New[int]("run")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 1, nil
	})

	e.RunIf(func() (bool, error) {
		runIf = true
		return false, nil
	})

	e.BeforeRun(func() error {
		t.Errorf("did not expect BeforeRun callback to run")
		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !runIf {
		t.Errorf("expected RunIf callback to run")
	}
}

func TestExperimentEmptyRunBefore(t *testing.T) {
	runIf := false

	e := New[int]("run")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})

	e.RunIf(func() (bool, error) {
		runIf = true
		return true, nil
	})

	e.BeforeRun(func() error {
		t.Errorf("did not expect BeforeRun callback to run")
		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !runIf {
		t.Errorf("expected RunIf callback to run")
	}
}

func TestExperimentRunIfError(t *testing.T) {
	e := New[int]("run")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})

	e.Try(func(ctx context.Context) (any, error) {
		t.Errorf("did not expect to run experiment if RunIf() returns error")
		return 1, nil
	})

	e.Publish(func(r *Result[int]) error {
		t.Errorf("did not expect to publish")
		return nil
	})

	e.ReportErrors(func(errors ...ResultError) {
		t.Errorf("did not expect to report errors")
	})

	e.RunIf(func() (bool, error) {
		return true, fmt.Errorf("run_if")
	})

	v, err := e.Run(context.Background())
	if v != *new(int) {
		t.Errorf("unexpected result: %v", v)
	}

	if err == nil {
		t.Errorf("expected a run_if error!")
	} else if err.Error() != "run_if" {
		t.Errorf("unexpected error: %v", err.Error())
	}
}

func TestExperimentSkipCompareMismatchedValues(t *testing.T) {
	e := New[int]("ignore")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 2, nil
	})
	e.Compare(func(control int, candidate interface{}) (bool, error) {
		return true, nil
	})

	published := false
	e.Publish(func(r *Result[int]) error {
		published = true

		if r.IsMismatched() {
			t.Errorf("Should not be matching")
		}

		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !published {
		t.Errorf("results never published")
	}
}

func TestExperimentSkipCompareMismatchedErrors(t *testing.T) {
	e := New[int]("ignore")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 1, errors.New("try")
	})
	e.Compare(func(control int, candidate interface{}) (bool, error) {
		return true, nil
	})

	published := false
	e.Publish(func(r *Result[int]) error {
		published = true

		if r.IsMatched() {
			t.Errorf("Should be mismatched")
		}

		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !published {
		t.Errorf("results never published")
	}
}

func TestExperimentSkipCompareSameErrors(t *testing.T) {
	e := New[int]("ignore")
	e.Synchronous = true
	e.Use(func(ctx context.Context) (int, error) {
		return 1, errors.New("ok")
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 1, errors.New("ok")
	})
	e.Compare(func(control int, candidate interface{}) (bool, error) {
		return true, nil
	})

	published := false
	e.Publish(func(r *Result[int]) error {
		published = true

		if r.IsMismatched() {
			t.Errorf("Should be matching")
		}

		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err == nil || err.Error() != "ok" {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !published {
		t.Errorf("results never published")
	}
}

func TestExperimentWithoutWaitingForCandidates(t *testing.T) {
	e := New[int]("ignore")
	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})

	wg := sync.WaitGroup{}
	wg.Add(1)

	candidateRun := false
	e.Try(func(ctx context.Context) (any, error) {
		time.Sleep(50 * time.Millisecond)
		candidateRun = true
		return 2, nil
	})

	published := false
	e.Publish(func(r *Result[int]) error {
		defer wg.Done()
		published = true
		return nil
	})

	v, err := e.Run(context.Background())
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if candidateRun {
		t.Errorf("Candidate run before expected")
	}

	wg.Wait()

	if !published {
		t.Errorf("results never published")
	}
}
