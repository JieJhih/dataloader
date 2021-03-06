package dataloader

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

///////////////////////////////////////////////////
// Tests
///////////////////////////////////////////////////
func TestLoader(t *testing.T) {
	t.Run("test Load method", func(t *testing.T) {
		t.Parallel()
		identityLoader, _ := IDLoader(0)
		ctx := context.Background()
		future := identityLoader.Load(ctx, "1")
		value, err := future()
		if err != nil {
			t.Error(err.Error())
		}
		if value != "1" {
			t.Error("load didn't return the right value")
		}
	})

	t.Run("test thunk does not contain race conditions", func(t *testing.T) {
		t.Parallel()
		identityLoader, _ := IDLoader(0)
		ctx := context.Background()
		future := identityLoader.Load(ctx, "1")
		go future()
		go future()
	})

	t.Run("test Load Method Panic Safety", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r != nil {
				t.Error("Panic Loader's panic should have been handled'")
			}
		}()
		panicLoader, _ := PanicLoader(0)
		ctx := context.Background()
		future := panicLoader.Load(ctx, "1")
		_, err := future()
		if err == nil || err.Error() != "Panic received in batch function: Programming error" {
			t.Error("Panic was not propagated as an error.")
		}
	})

	t.Run("test Load Method Panic Safety in multiple keys", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r != nil {
				t.Error("Panic Loader's panic should have been handled'")
			}
		}()
		panicLoader, _ := PanicLoader(0)
		futures := []Thunk{}
		ctx := context.Background()
		for i := 0; i < 3; i++ {
			futures = append(futures, panicLoader.Load(ctx, strconv.Itoa(i)))
		}
		for _, f := range futures {
			_, err := f()
			if err == nil || err.Error() != "Panic received in batch function: Programming error" {
				t.Error("Panic was not propagated as an error.")
			}
		}
	})

	t.Run("test LoadMany returns errors", func(t *testing.T) {
		t.Parallel()
		errorLoader, _ := ErrorLoader(0)
		ctx := context.Background()
		future := errorLoader.LoadMany(ctx, []interface{}{"1", "2", "3"})
		_, err := future()
		if len(err) != 3 {
			t.Error("LoadMany didn't return right number of errors")
		}
	})

	t.Run("test LoadMany returns len(errors) == len(keys)", func(t *testing.T) {
		t.Parallel()
		loader, _ := OneErrorLoader(3)
		ctx := context.Background()
		future := loader.LoadMany(ctx, []interface{}{"1", "2", "3"})
		_, err := future()
		if len(err) != 3 {
			t.Errorf("LoadMany didn't return right number of errors (should match size of input)")
		}

		if err[0] == nil {
			t.Error("Expected an error on the first item loaded")
		}

		if err[1] != nil || err[2] != nil {
			t.Error("Expected second and third errors to be nil")
		}
	})

	t.Run("test LoadMany returns nil []error when no errors occurred", func(t *testing.T) {
		t.Parallel()
		loader, _ := IDLoader(0)
		ctx := context.Background()
		_, err := loader.LoadMany(ctx, []interface{}{"1", "2", "3"})()
		if err != nil {
			t.Errorf("Expected LoadMany() to return nil error slice when no errors occurred")
		}
	})

	t.Run("test thunkmany does not contain race conditions", func(t *testing.T) {
		t.Parallel()
		identityLoader, _ := IDLoader(0)
		ctx := context.Background()
		future := identityLoader.LoadMany(ctx, []interface{}{"1", "2", "3"})
		go future()
		go future()
	})

	t.Run("test Load Many Method Panic Safety", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r != nil {
				t.Error("Panic Loader's panic should have been handled'")
			}
		}()
		panicLoader, _ := PanicLoader(0)
		ctx := context.Background()
		future := panicLoader.LoadMany(ctx, []interface{}{"1"})
		_, errs := future()
		if len(errs) < 1 || errs[0].Error() != "Panic received in batch function: Programming error" {
			t.Error("Panic was not propagated as an error.")
		}
	})

	t.Run("test LoadMany method", func(t *testing.T) {
		t.Parallel()
		identityLoader, _ := IDLoader(0)
		ctx := context.Background()
		future := identityLoader.LoadMany(ctx, []interface{}{"1", "2", "3"})
		results, _ := future()
		if results[0].(string) != "1" || results[1].(string) != "2" || results[2].(string) != "3" {
			t.Error("loadmany didn't return the right value")
		}
	})

	t.Run("batches many requests", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := IDLoader(0)
		ctx := context.Background()
		future1 := identityLoader.Load(ctx, "1")
		future2 := identityLoader.Load(ctx, "2")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future2()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1", "2"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not call batchFn in right order. Expected %#v, got %#v", expected, calls)
		}
	})

	t.Run("number of results matches number of keys", func(t *testing.T) {
		t.Parallel()
		faultyLoader, _ := FaultyLoader()
		ctx := context.Background()

		n := 10
		reqs := []Thunk{}
		keys := []interface{}{}
		for i := 0; i < n; i++ {
			key := strconv.Itoa(i)
			reqs = append(reqs, faultyLoader.Load(ctx, key))
			keys = append(keys, key)
		}

		for _, future := range reqs {
			_, err := future()
			if err == nil {
				t.Error("if number of results doesn't match keys, all keys should contain error")
			}
		}

		// TODO: expect to get some kind of warning
	})

	t.Run("responds to max batch size", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := IDLoader(2)
		ctx := context.Background()
		future1 := identityLoader.Load(ctx, "1")
		future2 := identityLoader.Load(ctx, "2")
		future3 := identityLoader.Load(ctx, "3")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future2()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future3()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner1 := []interface{}{"1", "2"}
		inner2 := []interface{}{"3"}
		expected := [][]interface{}{inner1, inner2}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not respect max batch size. Expected %#v, got %#v", expected, calls)
		}
	})

	t.Run("caches repeated requests", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := IDLoader(0)
		ctx := context.Background()
		future1 := identityLoader.Load(ctx, "1")
		future2 := identityLoader.Load(ctx, "1")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future2()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not respect max batch size. Expected %#v, got %#v", expected, calls)
		}
	})

	t.Run("allows primed cache", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := IDLoader(0)
		ctx := context.Background()
		identityLoader.Prime(ctx, "A", "Cached")
		future1 := identityLoader.Load(ctx, "1")
		future2 := identityLoader.Load(ctx, "A")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		value, err := future2()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not respect max batch size. Expected %#v, got %#v", expected, calls)
		}

		if value.(string) != "Cached" {
			t.Errorf("did not use primed cache value. Expected '%#v', got '%#v'", "Cached", value)
		}
	})

	t.Run("allows clear value in cache", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := IDLoader(0)
		ctx := context.Background()
		identityLoader.Prime(ctx, "A", "Cached")
		identityLoader.Prime(ctx, "B", "B")
		future1 := identityLoader.Load(ctx, "1")
		future2 := identityLoader.Clear(ctx, "A").Load(ctx, "A")
		future3 := identityLoader.Load(ctx, "B")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		value, err := future2()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future3()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1", "A"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not respect max batch size. Expected %#v, got %#v", expected, calls)
		}

		if value.(string) != "A" {
			t.Errorf("did not use primed cache value. Expected '%#v', got '%#v'", "Cached", value)
		}
	})

	t.Run("allows clearAll values in cache", func(t *testing.T) {
		t.Parallel()
		batchOnlyLoader, loadCalls := BatchOnlyLoader(0)
		ctx := context.Background()
		future1 := batchOnlyLoader.Load(ctx, "1")
		future2 := batchOnlyLoader.Load(ctx, "1")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future2()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not batch queries. Expected %#v, got %#v", expected, calls)
		}

		if _, found := batchOnlyLoader.cache.Get(ctx, "1"); found {
			t.Errorf("did not clear cache after batch. Expected %#v, got %#v", false, found)
		}
	})

	t.Run("allows clearAll values in cache", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := IDLoader(0)
		ctx := context.Background()
		identityLoader.Prime(ctx, "A", "Cached")
		identityLoader.Prime(ctx, "B", "B")

		identityLoader.ClearAll()

		future1 := identityLoader.Load(ctx, "1")
		future2 := identityLoader.Load(ctx, "A")
		future3 := identityLoader.Load(ctx, "B")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future2()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future3()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1", "A", "B"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not respect max batch size. Expected %#v, got %#v", expected, calls)
		}
	})

	t.Run("all methods on NoCache are Noops", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := NoCacheLoader(0)
		ctx := context.Background()
		identityLoader.Prime(ctx, "A", "Cached")
		identityLoader.Prime(ctx, "B", "B")

		identityLoader.ClearAll()

		future1 := identityLoader.Clear(ctx, "1").Load(ctx, "1")
		future2 := identityLoader.Load(ctx, "A")
		future3 := identityLoader.Load(ctx, "B")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future2()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future3()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1", "A", "B"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not respect max batch size. Expected %#v, got %#v", expected, calls)
		}
	})

	t.Run("no cache does not cache anything", func(t *testing.T) {
		t.Parallel()
		identityLoader, loadCalls := NoCacheLoader(0)
		ctx := context.Background()
		identityLoader.Prime(ctx, "A", "Cached")
		identityLoader.Prime(ctx, "B", "B")

		future1 := identityLoader.Load(ctx, "1")
		future2 := identityLoader.Load(ctx, "A")
		future3 := identityLoader.Load(ctx, "B")

		_, err := future1()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future2()
		if err != nil {
			t.Error(err.Error())
		}
		_, err = future3()
		if err != nil {
			t.Error(err.Error())
		}

		calls := *loadCalls
		inner := []interface{}{"1", "A", "B"}
		expected := [][]interface{}{inner}
		if !reflect.DeepEqual(calls, expected) {
			t.Errorf("did not respect max batch size. Expected %#v, got %#v", expected, calls)
		}
	})

}

// test helpers
func IDLoader(max int) (*Loader, *[][]interface{}) {
	var mu sync.Mutex
	var loadCalls [][]interface{}
	identityLoader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		var results []*Result
		mu.Lock()
		loadCalls = append(loadCalls, keys)
		mu.Unlock()
		for _, key := range keys {
			results = append(results, &Result{key, nil})
		}
		return results
	}, WithBatchCapacity(max))
	return identityLoader, &loadCalls
}
func BatchOnlyLoader(max int) (*Loader, *[][]interface{}) {
	var mu sync.Mutex
	var loadCalls [][]interface{}
	identityLoader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		var results []*Result
		mu.Lock()
		loadCalls = append(loadCalls, keys)
		mu.Unlock()
		for _, key := range keys {
			results = append(results, &Result{key, nil})
		}
		return results
	}, WithBatchCapacity(max), WithClearCacheOnBatch())
	return identityLoader, &loadCalls
}
func ErrorLoader(max int) (*Loader, *[][]interface{}) {
	var mu sync.Mutex
	var loadCalls [][]interface{}
	identityLoader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		var results []*Result
		mu.Lock()
		loadCalls = append(loadCalls, keys)
		mu.Unlock()
		for _, key := range keys {
			results = append(results, &Result{key, fmt.Errorf("this is a test error")})
		}
		return results
	}, WithBatchCapacity(max))
	return identityLoader, &loadCalls
}
func OneErrorLoader(max int) (*Loader, *[][]interface{}) {
	var mu sync.Mutex
	var loadCalls [][]interface{}
	identityLoader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		results := make([]*Result, max)
		mu.Lock()
		loadCalls = append(loadCalls, keys)
		mu.Unlock()
		for i, key := range keys {
			var err error
			if i == 0 {
				err = errors.New("always error on the first key")
			}
			results[i] = &Result{key, err}
		}
		return results
	}, WithBatchCapacity(max))
	return identityLoader, &loadCalls
}
func PanicLoader(max int) (*Loader, *[][]interface{}) {
	var loadCalls [][]interface{}
	panicLoader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		panic("Programming error")
	}, WithBatchCapacity(max), withSilentLogger())
	return panicLoader, &loadCalls
}
func BadLoader(max int) (*Loader, *[][]interface{}) {
	var mu sync.Mutex
	var loadCalls [][]interface{}
	identityLoader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		var results []*Result
		mu.Lock()
		loadCalls = append(loadCalls, keys)
		mu.Unlock()
		results = append(results, &Result{keys[0], nil})
		return results
	}, WithBatchCapacity(max))
	return identityLoader, &loadCalls
}
func NoCacheLoader(max int) (*Loader, *[][]interface{}) {
	var mu sync.Mutex
	var loadCalls [][]interface{}
	cache := &NoCache{}
	identityLoader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		var results []*Result
		mu.Lock()
		loadCalls = append(loadCalls, keys)
		mu.Unlock()
		for _, key := range keys {
			results = append(results, &Result{key, nil})
		}
		return results
	}, WithCache(cache), WithBatchCapacity(max))
	return identityLoader, &loadCalls
}

// FaultyLoader gives len(keys)-1 results.
func FaultyLoader() (*Loader, *[][]interface{}) {
	var mu sync.Mutex
	var loadCalls [][]interface{}

	loader := NewBatchedLoader(func(_ context.Context, keys []interface{}) []*Result {
		var results []*Result
		mu.Lock()
		loadCalls = append(loadCalls, keys)
		mu.Unlock()

		lastKeyIndex := len(keys) - 1
		for i, key := range keys {
			if i == lastKeyIndex {
				break
			}

			results = append(results, &Result{key, nil})
		}
		return results
	})

	return loader, &loadCalls
}

///////////////////////////////////////////////////
// Benchmarks
///////////////////////////////////////////////////
var a = &Avg{}

func batchIdentity(_ context.Context, keys []interface{}) (results []*Result) {
	a.Add(len(keys))
	for _, key := range keys {
		results = append(results, &Result{key, nil})
	}
	return
}

var _ctx context.Context = context.Background()

func BenchmarkLoader(b *testing.B) {
	UserLoader := NewBatchedLoader(batchIdentity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UserLoader.Load(_ctx, strconv.Itoa(i))
	}
	log.Printf("avg: %f", a.Avg())
}

type Avg struct {
	total  float64
	length float64
	lock   sync.RWMutex
}

func (a *Avg) Add(v int) {
	a.lock.Lock()
	a.total += float64(v)
	a.length++
	a.lock.Unlock()
}

func (a *Avg) Avg() float64 {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if a.total == 0 {
		return 0
	} else if a.length == 0 {
		return 0
	}
	return a.total / a.length
}
