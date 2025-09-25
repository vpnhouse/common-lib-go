package xttlmap

import (
	"sync"
	"testing"
	"time"
)

func deadline(ttl time.Duration) time.Time {
	t := time.Now().Add(ttl)
	return t
}

func TestSetGet(t *testing.T) {
	store := New[string, string](3)

	store.Set("key1", "value1", deadline(time.Minute))
	store.Set("key2", "value2", deadline(time.Minute))

	if val, ok := store.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	if val, ok := store.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	if _, ok := store.Get("key3"); ok {
		t.Error("Expected key3 to not exist")
	}
}

func TestMaxSize(t *testing.T) {
	store := New[string, string](2)

	store.Set("key1", "value1", deadline(time.Minute))
	store.Set("key2", "value2", deadline(time.Minute))

	if val, ok := store.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	if val, ok := store.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	if _, ok := store.Get("key3"); ok {
		t.Error("Expected key3 to not exist")
	}

	store.Set("key3", "value3", deadline(time.Minute))
	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to not exist")
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	if val, ok := store.Get("key3"); !ok || val != "value3" {
		t.Errorf("Expected value3, got %v", val)
	}
	if val, ok := store.Get("key3"); !ok || val != "value3" {
		t.Errorf("Expected value3, got %v", val)
	}
}

func TestExpiration(t *testing.T) {
	store := New[string, string](100)

	store.Set("key1", "value1", deadline(100*time.Millisecond))
	store.Set("key2", "value2", deadline(200*time.Millisecond))

	time.Sleep(50 * time.Millisecond)
	if val, ok := store.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}

	time.Sleep(200 * time.Millisecond)

	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to be expired")
	}
	if _, ok := store.Get("key2"); ok {
		t.Error("Expected key2 to be expired")
	}
}

func TestDelete(t *testing.T) {
	store := New[string, string](100)

	store.Set("key1", "value1", deadline(time.Minute))
	store.Set("key2", "value2", deadline(time.Minute))

	store.Delete("key1")
	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to be deleted")
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := New[string, int](1000)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			store.Set("key", i, deadline(time.Minute))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			store.Get("key")
		}
	}()

	wg.Wait()
}

func TestCleanup(t *testing.T) {
	store := New[string, string](100)

	store.Set("key1", "value1", deadline(100*time.Millisecond))
	store.Set("key2", "value2", deadline(200*time.Millisecond))

	time.Sleep(300 * time.Millisecond)

	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to be expired")
	}
	if _, ok := store.Get("key2"); ok {
		t.Error("Expected key2 to be expired")
	}
}

func TestResize(t *testing.T) {
	store := New[string, string](3)

	store.Set("key1", "value1", deadline(100*time.Millisecond))
	store.Set("key2", "value2", deadline(200*time.Millisecond))

	store.Resize(1)

	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to be removed")
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
}
