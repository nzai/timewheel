package timewheel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSetAndExpire(t *testing.T) {
	var wg sync.WaitGroup
	tw := NewTimeWheel(100*time.Millisecond, 10, func(k string, v any) {
		wg.Done()
	})

	wg.Add(1)
	start := time.Now()
	tw.Set("test", "data", 300*time.Millisecond)

	wg.Wait()

	elapsed := time.Since(start)
	if elapsed > 301*time.Millisecond {
		t.Errorf("Expected callback to occur within 300ms, but took %s", elapsed)
	}
}

func TestDelete(t *testing.T) {
	called := false
	tw := NewTimeWheel(100*time.Millisecond, 10, func(k string, v any) {
		called = true
	})

	tw.Set("test", "data", 200*time.Millisecond)
	tw.Delete("test")
	time.Sleep(300 * time.Millisecond)

	if called {
		t.Error("Callback should not be called after deletion")
	}
}

func TestMove(t *testing.T) {
	callCount := 0
	tw := NewTimeWheel(100*time.Millisecond, 10, func(string, any) {
		callCount++
	})

	tw.Set("test", "data", 200*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	tw.Move("test", 200*time.Millisecond)
	time.Sleep(250 * time.Millisecond)

	if callCount != 1 {
		t.Errorf("Expected 1 callback, got %d", callCount)
	}
}

func TestFlushAll(t *testing.T) {
	called := false
	tw := NewTimeWheel(100*time.Millisecond, 10, func(string, any) {
		called = true
	})

	tw.Set("test1", "data", 100*time.Millisecond)
	tw.Set("test2", "data", 200*time.Millisecond)
	tw.FlushAll()
	time.Sleep(300 * time.Millisecond)

	if called {
		t.Error("No callbacks should occur after flush")
	}
}

func TestConcurrentAccess(t *testing.T) {
	tw := NewTimeWheel(10*time.Millisecond, 100, func(string, any) {})

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key_%d", n)
			tw.Set(key, n, time.Duration(n%100)*time.Millisecond)
			if n%2 == 0 {
				tw.Delete(key)
			}
		}(i)
	}
	wg.Wait()
}
