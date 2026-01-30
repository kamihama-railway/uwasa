package uwasa

import (
	"testing"
	"fmt"
)

func TestNeoExUnsafeSafety(t *testing.T) {
	// Test stack overflow protection
	depth := 100
	expr := "a"
	for i := 0; i < depth; i++ {
		expr = fmt.Sprintf("a + (%s)", expr)
	}
	engine, err := NewEngineVMNeo(expr)
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	_, err = engine.Execute(map[string]any{"a": int64(1)})
	if err == nil {
		t.Errorf("Expected error for deep stack, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	// Test concurrent access (Engine should be thread-safe as VM uses local stack)
	engine2, _ := NewEngineVMNeo("a + b")
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("Concurrent_%d", i), func(t *testing.T) {
			t.Parallel()
			vars := map[string]any{"a": int64(i), "b": int64(10)}
			res, err := engine2.Execute(vars)
			if err != nil {
				t.Errorf("Execute failed: %v", err)
			}
			if res != int64(i + 10) {
				t.Errorf("Expected %d, got %v", i+10, res)
			}
		})
	}
}

func TestNeoExUnsafe_GetGlobalNil(t *testing.T) {
    engine, _ := NewEngineVMNeo("a + 1")
    // Missing "a" in vars
    res, err := engine.Execute(map[string]any{"b": 1})
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }
    // Result depends on implementation of nil + 1, but it shouldn't crash.
    t.Logf("Result of nil + 1: %v", res)
}
