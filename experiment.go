package scientist

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
	"time"
)

var ErrorOnMismatches bool

func New[T any](name string) *Experiment[T] {
	return &Experiment[T]{
		Name:          name,
		Context:       make(map[string]string),
		behaviors:     []*behavior[any]{},
		comparator:    defaultComparator[T],
		runcheck:      defaultRunCheck,
		publisher:     defaultPublisher[T],
		errorReporter: defaultErrorReporter,
		beforeRun:     defaultBeforeRun,
		cleaner:       defaultCleaner,
	}
}

type behavior[T any] struct {
	name string
	fn   func(context.Context) (T, error)
}

type Experiment[T any] struct {
	Name        string
	Context     map[string]string
	Synchronous bool

	control       *behavior[T]
	behaviors     []*behavior[any]
	ignores       []func(control T, candidate any) (bool, error)
	comparator    func(control T, candidate any) (bool, error)
	runcheck      func() (bool, error)
	publisher     func(*Result[T]) error
	errorReporter func(...ResultError)
	beforeRun     func() error
	cleaner       func(any) (any, error)
}

func (e *Experiment[T]) Use(fn func(ctx context.Context) (T, error)) {
	e.control = &behavior[T]{name: controlBehavior, fn: fn}
}

func (e *Experiment[T]) Try(fn func(ctx context.Context) (any, error)) {
	e.Behavior(candidateBehavior, fn)
}

func (e *Experiment[T]) Behavior(name string, fn func(ctx context.Context) (any, error)) {
	e.behaviors = append(e.behaviors, &behavior[any]{name: name, fn: fn})
}

func (e *Experiment[T]) Compare(fn func(control T, candidate any) (bool, error)) {
	e.comparator = fn
}

func (e *Experiment[T]) Clean(fn func(v any) (interface{}, error)) {
	e.cleaner = fn
}

func (e *Experiment[T]) Ignore(fn func(control T, candidate any) (bool, error)) {
	e.ignores = append(e.ignores, fn)
}

func (e *Experiment[T]) RunIf(fn func() (bool, error)) {
	e.runcheck = fn
}

func (e *Experiment[T]) BeforeRun(fn func() error) {
	e.beforeRun = fn
}

func (e *Experiment[T]) Publish(fn func(*Result[T]) error) {
	e.publisher = fn
}

func (e *Experiment[T]) ReportErrors(fn func(...ResultError)) {
	e.errorReporter = fn
}

func (e *Experiment[T]) isEnabled() (bool, error) {
	if e.control == nil {
		return false, behaviorNotFound(e, controlBehavior)
	}

	return e.runcheck()
}

func (e *Experiment[T]) Run(ctx context.Context) (T, error) {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[scientist] stacktrace from unhandled exception:%v Stacktrace: %s", err, string(debug.Stack()))
		}
	}()

	enabled, err := e.isEnabled()
	if err != nil {
		return *new(T), err
	}

	control := observe(ctx, e, e.control)
	if enabled && len(e.behaviors) > 0 {
		r := &Result[T]{
			Experiment: e,
			Control:    control,
		}

		if e.Synchronous {
			e.run(ctx, r)
		} else {
			go e.run(ctx, r)
		}
	}

	return control.Value, control.Err
}

func (e *Experiment[T]) run(ctx context.Context, r *Result[T]) {
	defer func() {
		r.finalize()

		if err := e.publisher(r); err != nil {
			r.addError("publish", err)
		}

		if len(r.Errors) > 0 {
			e.errorReporter(r.Errors...)
		}
	}()

	if err := e.beforeRun(); err != nil {
		r.addError("before_run", err)
		return
	}

	r.Candidates = make([]*Observation[T, any], 0, len(e.behaviors))

	var wg sync.WaitGroup
	wg.Add(len(e.behaviors))
	finished := make(chan *Observation[T, any], len(e.behaviors))
	go func() {
		wg.Wait()
		close(finished)
	}()

	for _, b := range shuffle(e.behaviors) {
		go func(ctx context.Context, b *behavior[any]) {
			defer wg.Done()
			finished <- observe(ctx, e, b)
		}(context.WithoutCancel(ctx), b)
	}

	for candidate := range finished {
		r.Candidates = append(r.Candidates, candidate)
	}
}

// https://www.calhoun.io/using-named-return-variables-to-capture-panics-in-go/
func observe[TE any, TB any](ctx context.Context, e *Experiment[TE], b *behavior[TB]) *Observation[TE, TB] {
	o := &Observation[TE, TB]{
		Experiment: e,
		Name:       b.name,
		Started:    time.Now(),
	}

	defer func() {
		if r := recover(); r != nil {
			o.Err = fmt.Errorf("recover from bad behavior %s: %v", b.name, r)
		}
	}()

	v, err := b.fn(ctx)
	o.Runtime = time.Since(o.Started)
	o.Value = v
	o.Err = err

	return o
}

func defaultComparator[T any](candidate T, control any) (bool, error) {
	return reflect.DeepEqual(candidate, control), nil
}

func defaultRunCheck() (bool, error) {
	return true, nil
}

func defaultCleaner(v any) (any, error) {
	return v, nil
}

func defaultPublisher[T any](r *Result[T]) error {
	return nil
}

func defaultErrorReporter(errs ...ResultError) {
	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "[scientist] error during %q for %q experiment: (%T) %v\n", err.Operation, err.Experiment, err.Err, err.Err)
	}
}

func defaultBeforeRun() error {
	return nil
}

func behaviorNotFound[T any](e *Experiment[T], name string) error {
	return fmt.Errorf("Behavior %q not found for experiment %q", name, e.Name)
}
