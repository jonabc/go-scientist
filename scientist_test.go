package scientist

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func basicExperiment(e *Experiment[int]) {
	e.Synchronous = true

	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})

	e.Try(func(ctx context.Context) (any, error) {
		return 2, nil
	})

	e.Behavior("three", func(ctx context.Context) (any, error) {
		return 3, nil
	})

	e.Behavior("correct", func(ctx context.Context) (any, error) {
		return 1, nil
	})
}

func TestRun(t *testing.T) {
	r, err := Run(context.Background(), "test", func(e *Experiment[int]) error {
		basicExperiment(e)
		e.Publish(func(r *Result[int]) error {
			if len(r.Errors) != 0 {
				t.Errorf("Unexpected experiment errors: %v", r.Errors)
			}

			if r.Control.Name != "control" {
				t.Errorf("Unexpected control observation name: %q", r.Control.Name)
			}

			if r.Control.Err != nil {
				t.Errorf("Expected no error, got: %v", r.Control.Err)
			}

			if r.Control.Value != 1 {
				t.Errorf("Bad value for 'control': %v", r.Control.Value)
			}

			assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
			assertObservationNames(t, "ignored", r.Ignored, []string{})
			assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate", "three"})

			candidatesMap := make(map[string]*Observation[int, any], len(r.Candidates))
			for _, o := range r.Candidates {
				candidatesMap[o.Name] = o
			}

			two, ok := candidatesMap["candidate"]
			if !ok {
				t.Errorf("No behavior 'candidate'")
			} else {
				if two.Err != nil {
					t.Errorf("Error for 'candidate': %v", two.Err)
				}

				if two.Value != 2 {
					t.Errorf("Bad value for 'candidate': %v", two.Value)
				}
			}

			three, ok := candidatesMap["three"]
			if !ok {
				t.Errorf("No behavior 'three'")
			} else {
				if three.Err != nil {
					t.Errorf("Error for 'three': %v", three.Err)
				}

				if three.Value != 3 {
					t.Errorf("Bad value for 'three': %v", three.Value)
				}
			}

			correct, ok := candidatesMap["correct"]
			if !ok {
				t.Errorf("No behavior 'correct'")
			} else {
				if correct.Err != nil {
					t.Errorf("Error for 'correct': %v", correct.Err)
				}

				if correct.Value != 1 {
					t.Errorf("Bad value for 'correct': %v", correct.Value)
				}
			}

			return nil
		})
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if r != 1 {
		t.Errorf("Unexpected result: %v", r)
	}
}

func TestIgnore(t *testing.T) {
	r, err := Run(context.Background(), "testIgnore", func(e *Experiment[int]) error {
		basicExperiment(e)
		e.Ignore(func(control int, candidate any) (bool, error) {
			return candidate == 3, nil
		})
		e.Publish(func(r *Result[int]) error {
			if len(r.Errors) != 0 {
				t.Errorf("Unexpected experiment errors: %v", r.Errors)
			}

			assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
			assertObservationNames(t, "ignored", r.Ignored, []string{"three"})
			assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate"})
			return nil
		})
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if r != 1 {
		t.Errorf("Unexpected result: %v", r)
	}
}

func TestCompare(t *testing.T) {
	r, err := Run(context.Background(), "testCompare", func(e *Experiment[int]) error {
		basicExperiment(e)
		e.Compare(func(control int, candidate any) (bool, error) {
			return control == 1 && candidate == 3, nil
		})

		e.Publish(func(r *Result[int]) error {
			if len(r.Errors) != 0 {
				t.Errorf("Unexpected experiment errors: %v", r.Errors)
			}

			assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
			assertObservationNames(t, "ignored", r.Ignored, []string{})
			assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate", "correct"})
			return nil
		})

		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if r != 1 {
		t.Errorf("Unexpected result: %v", r)
	}
}

func TestCompareAndIgnore(t *testing.T) {
	r, err := Run(context.Background(), "testCompareAndIgnore", func(e *Experiment[int]) error {
		basicExperiment(e)
		e.Compare(func(control int, candidate any) (bool, error) {
			return control == 1 && candidate == 3, nil
		})
		e.Ignore(func(control int, candidate any) (bool, error) {
			return candidate == 1, nil
		})
		e.Publish(func(r *Result[int]) error {
			if len(r.Errors) != 0 {
				t.Errorf("Unexpected experiment errors: %v", r.Errors)
			}

			assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
			assertObservationNames(t, "ignored", r.Ignored, []string{"correct"})
			assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate"})
			return nil
		})
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if r != 1 {
		t.Errorf("Unexpected result: %v", r)
	}
}

func TestDefaultCleaner(t *testing.T) {
	_, err := Run(context.Background(), "cleaner", func(e *Experiment[string]) error {
		e.Use(func(ctx context.Context) (string, error) {
			return "booya", nil
		})
		e.Publish(func(r *Result[string]) error {
			cleaned, err := r.Control.CleanedValue()
			if err != nil {
				t.Errorf("Unexpected cleaning error: %v", err)
			}

			if cleaned != "booya" {
				t.Errorf("bad cleaned value: %v", cleaned)
			}
			return nil
		})
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCustomCleaner(t *testing.T) {
	_, err := Run(context.Background(), "cleaner", func(e *Experiment[string]) error {
		e.Use(func(ctx context.Context) (string, error) {
			return "booya", nil
		})
		e.Clean(func(v any) (any, error) {
			return strings.ToUpper(v.(string)), nil
		})
		e.Publish(func(r *Result[string]) error {
			cleaned, err := r.Control.CleanedValue()
			if err != nil {
				t.Errorf("Unexpected cleaning error: %v", err)
			}

			if cleaned != "BOOYA" {
				t.Errorf("bad cleaned value: %v", cleaned)
			}

			return nil
		})
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func assertObservationNames[T any](t *testing.T, key string, obs []*Observation[T, any], expected []string) {
	actual := observationNames(obs)
	if reflect.DeepEqual(expected, actual) {
		return
	}

	t.Errorf("Expected %s observations: %v, got: %v", key, expected, actual)
}

func observationNames[T any](obs []*Observation[T, any]) []string {
	names := make([]string, len(obs))
	for i, o := range obs {
		names[i] = o.Name
	}
	sort.Strings(names)
	return names
}
