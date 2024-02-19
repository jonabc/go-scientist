package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/freshworks/go-scientist"
)

var (
	arr = make([]int, 10000)
	set = make(map[int]bool, 10000)
)

func init() {
	for i := 0; i < 10000; i++ {
		arr[i] = i
		set[i] = true
	}
}

func main() {
	n := 9999
	controlFn := func(ctx context.Context) (bool, error) {
		for _, i := range arr {
			if i == n {
				return true, nil
			}
		}

		return false, nil
	}
	candidateFn := func(ctx context.Context) (any, error) {
		return set[n], nil
	}

	Run(controlFn, candidateFn)
	RunAsyncCandidates(controlFn, candidateFn)
}

func Run[T any](controlFn func(ctx context.Context) (T, error), candidateFn func(ctx context.Context) (any, error)) {
	start := time.Now()
	defer func() {
		fmt.Printf("Run experiment time elapsed: %s\n", time.Since(start))
	}()

	e := scientist.New[T]("synchronous")
	e.Synchronous = true
	e.Use(controlFn)
	e.Try(candidateFn)

	e.Context["control"] = "array"
	e.Context["candidate"] = "map"
	e.Context["run_type"] = "sync"

	e.Publish(publish)
	result, err := e.Run(context.Background())
	if err != nil {
		fmt.Printf("experiment error: %q\n", err)
		return
	}
	fmt.Printf("The arbitrary example returned: %v\n", result)
}

func RunAsyncCandidates[T any](controlFn func(ctx context.Context) (T, error), candidateFn func(ctx context.Context) (any, error)) {
	start := time.Now()
	defer func() {
		fmt.Printf("RunAsyncCandidates experiment time elapsed: %s\n", time.Since(start))
	}()

	e2 := scientist.New[T]("asynchronousCandidates")
	e2.Use(controlFn)
	e2.Try(candidateFn)

	e2.Context["control"] = "array"
	e2.Context["candidate"] = "map"
	e2.Context["run_type"] = "asyncCandidates"

	wg := sync.WaitGroup{}
	e2.Publish(func(r *scientist.Result[T]) error {
		defer wg.Done()
		return publish(r)
	})
	wg.Add(1)
	result, err := e2.Run(context.Background())
	if err != nil {
		fmt.Printf("experiment error: %q\n", err)
		return
	}
	wg.Wait()
	fmt.Printf("The arbitrary example returned: %v\n", result)
}

func publish[T any](r *scientist.Result[T]) error {
	fmt.Println("Experiment:", r.Experiment.Name)
	publishObservation(r.Control)
	for _, o := range r.Candidates {
		publishObservation(o)
	}
	fmt.Println(" context:")
	for key, value := range r.Experiment.Context {
		fmt.Printf("   %q: %q\n", key, value)
	}
	return nil
}

func publishObservation[TE any, TVal any](o *scientist.Observation[TE, TVal]) {
	fmt.Println(" observation", o)
	fmt.Printf("   name: %s\n", o.Name)
	fmt.Printf("   value: %v\n", o.Value)
	fmt.Printf("   err: %v\n", o.Err)
	fmt.Printf("   time: %v\n", o.Runtime)
}
