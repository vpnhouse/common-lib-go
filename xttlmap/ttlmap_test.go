package xttlmap

import (
	"sync"
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	store := New[string, string]()
	defer store.Stop()

	store.Set("key1", "value1", time.Minute)
	store.Set("key2", "value2", time.Minute)

	if val, ok := store.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	if _, ok := store.Get("key3"); ok {
		t.Error("Expected key3 to not exist")
	}
}

func TestExpiration(t *testing.T) {
	store := New[string, string]()
	defer store.Stop()

	store.Set("key1", "value1", 100*time.Millisecond)
	store.Set("key2", "value2", 200*time.Millisecond)

	// Проверяем, что значения доступны до истечения TTL
	time.Sleep(50 * time.Millisecond)
	if val, ok := store.Get("key1"); !ok || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}

	// Ждём, пока истечёт TTL для key1 и key2
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что записи истекли
	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to be expired")
	}
	if _, ok := store.Get("key2"); ok {
		t.Error("Expected key2 to be expired")
	}
}

func TestDelete(t *testing.T) {
	store := New[string, string]()
	defer store.Stop()

	store.Set("key1", "value1", time.Minute)
	store.Set("key2", "value2", time.Minute)

	store.Delete("key1")
	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to be deleted")
	}
	if val, ok := store.Get("key2"); !ok || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := New[string, int]()
	defer store.Stop()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			store.Set("key", i, time.Minute)
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
	store := New[string, string]()
	defer store.Stop()

	store.Set("key1", "value1", 100*time.Millisecond)
	store.Set("key2", "value2", 200*time.Millisecond)

	time.Sleep(300 * time.Millisecond)

	if _, ok := store.Get("key1"); ok {
		t.Error("Expected key1 to be expired")
	}
	if _, ok := store.Get("key2"); ok {
		t.Error("Expected key2 to be expired")
	}
}

func TestStop(t *testing.T) {
	store := New[string, string]()
	store.Set("key1", "value1", time.Minute)

	store.Stop()

	// Проверяем, что операции больше не выполняются
	if _, ok := store.Get("key1"); ok {
		t.Error("Expected store to be stopped")
	}
	store.Set("key2", "value2", time.Minute)
	if _, ok := store.Get("key2"); ok {
		t.Error("Expected store to be stopped")
	}
}
