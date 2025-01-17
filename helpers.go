package scientist

import (
	"fmt"
	"math/rand"
	"time"
)

func Bool(ok interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	switch t := ok.(type) {
	case bool:
		return t, nil
	default:
		return false, fmt.Errorf("[scientist] bad result type: %v (%T)", ok, ok)
	}
}

// Shuffle randomizes the behavior access.
func shuffle[T any](behaviors []T) []T {
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	arr := behaviors
	for i := len(arr) - 1; i > 0; i-- {
		j := r.Intn(i)
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}
