package runtimeowner

import (
	"context"
	"sync"
	"testing"

	"github.com/dop251/goja"
)

func TestRunnerConcurrentCallStress(t *testing.T) {
	vm := goja.New()
	s := newQueueScheduler(vm)
	defer s.Close()

	r := NewRunner(vm, s, Options{RecoverPanics: true})

	var (
		wg      sync.WaitGroup
		counter int
	)

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.Call(context.Background(), "stress.call", func(context.Context, *goja.Runtime) (any, error) {
				counter++
				return counter, nil
			})
			if err != nil {
				t.Errorf("call error: %v", err)
			}
		}()
	}

	wg.Wait()
	if counter != 500 {
		t.Fatalf("unexpected counter value: %d", counter)
	}
}
