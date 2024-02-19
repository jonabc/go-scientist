package scientist

import (
	"context"
	"fmt"
	"testing"
)

func TestPublish(t *testing.T) {
	e := New[int]("publish")
	e.Synchronous = true

	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 2, nil
	})

	published := false
	reported := false
	e.Publish(func(r *Result[int]) error {
		published = true

		if r.Experiment.Name != "publish" {
			t.Errorf("Bad experiment name: %q", r.Experiment.Name)
		}

		return nil
	})

	e.ReportErrors(func(errors ...ResultError) {
		reported = true
		t.Errorf("result errors reported :(")
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

	if reported {
		t.Errorf("result errors reported :(")
	}
}

func TestPublishWithErrors(t *testing.T) {
	e := New[int]("publish")
	e.Synchronous = true

	e.Use(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	e.Try(func(ctx context.Context) (any, error) {
		return 2, nil
	})

	e.Compare(func(control int, candidate any) (bool, error) {
		return true, fmt.Errorf("(compare) candidate: %d", candidate)
	})
	// ignore callback 0, no error
	e.Ignore(func(control int, candidate any) (bool, error) {
		return false, nil
	})
	// ignore callback 1, returns an error
	e.Ignore(func(control int, candidate any) (bool, error) {
		return true, fmt.Errorf("(ignore) candidate: %d", candidate)
	})

	published := false
	reported := make(map[string]int)
	e.Publish(func(r *Result[int]) error {
		published = true
		return fmt.Errorf("(publish) result: %s", r.Experiment.Name)
	})

	e.ReportErrors(func(errors ...ResultError) {
		for _, err := range errors {
			reported[err.Operation] = reported[err.Operation] + 1
			if err.Experiment != e.Name {
				t.Errorf("Bad experiment name for %q operation: %q", err.Operation, err.Experiment)
			}
			switch err.Operation {
			case "compare":
				if actual := err.Error(); actual != "(compare) candidate: 2" {
					t.Errorf("Bad error message for compare operation: %q", actual)
				}
			case "ignore":
				if actual := err.Error(); actual != "(ignore) candidate: 2" {
					t.Errorf("Bad error message for ignore operation: %q", actual)
				}
			case "publish":
				if actual := err.Error(); actual != "(publish) result: publish" {
					t.Errorf("Bad error message for publish operation: %q", actual)
				}
			default:
				t.Errorf("Bad operation: %q", err.Operation)
			}
		}
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

	if len(reported) != 3 {
		t.Errorf("all result errors not reported: %v", reported)
	}

	for key, times := range reported {
		if times != 1 {
			t.Errorf("%q errors reported %d times", key, times)
		}
	}
}
