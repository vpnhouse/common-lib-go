package shaper

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTimeBucket_BasicFunctionality(t *testing.T) {
	ctx := context.Background()
	//1 MiBps speed, 1MiB burst
	shaper := NewTimeBucket(ctx, 1024, 1024)

	start := time.Now()

	for idx := 0; idx < 10; idx++ {
		result := shaper.Shape(1024 * 200)

		if !result {
			t.Error("Shape returned false, expected true")
		}

	}

	elapsed := time.Since(start)

	// Nearly 2 MiB transfererd, first 1 MiB is immediately due to burst, 2nd one is delayed for 1 second
	expected := time.Second
	if elapsed < expected/2 || elapsed > expected*2 {
		t.Errorf("Expected ~1ms, got %v", elapsed)
	}
}

func TestTimeBucket_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	shaper := NewTimeBucket(ctx, 1024, 1024)

	// First run should immediately return
	if !shaper.Shape(1024) {
		t.Error("First Shape should succeed")
	}

	cancel()

	// Large enouogh not to fit into burst
	if shaper.Shape(10 * 1024 * 1024) {
		t.Error("Shape should return false after context cancellation")
	}
}

func TestTimeBucket_Burst(t *testing.T) {
	fmt.Println("BURST")
	ctx := context.Background()

	// Slow shaper
	shaper := NewTimeBucket(ctx, 1, 10)
	start := time.Now()

	// Exhaust burst
	for i := 0; i < 10; i++ {
		if !shaper.Shape(1000) {
			t.Error("Burst packets should pass immediately")
		}
	}

	// Expected nearly immediatly
	burstTime := time.Since(start)
	if burstTime > 10*time.Millisecond {
		t.Errorf("Burst packets took too long: %v", burstTime)
	}

	// One more KiB to pass
	shapeStart := time.Now()
	if !shaper.Shape(1024) {
		t.Error("Shape after burst should return true")
	}

	// Expected to take one second
	shapeTime := time.Since(shapeStart)
	expected := time.Second
	if shapeTime < expected/2 || shapeTime > expected*2 {
		t.Errorf("Expected ~1s delay, got %v", shapeTime)
	}
}

func TestTimeBucket_ZeroLength(t *testing.T) {
	ctx := context.Background()
	shaper := NewTimeBucket(ctx, 1024, 1024)

	start := time.Now()
	if !shaper.Shape(0) {
		t.Error("Zero length should return true")
	}

	// Espected return immediately
	elapsed := time.Since(start)
	if elapsed > time.Millisecond {
		t.Errorf("Zero length should be instant, took %v", elapsed)
	}
}

func TestTimeBucket_NilSafety(t *testing.T) {
	var shaper *TimeBucket = nil

	// Should not panic
	result := shaper.Shape(1024)
	if !result {
		t.Error("Nil shaper should return true")
	}
}
