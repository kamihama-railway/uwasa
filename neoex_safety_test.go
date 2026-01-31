// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

func TestNeoExVM_Safety_EmptyBytecode(t *testing.T) {
	bc := &NeoBytecode{
		Instructions: nil,
		Constants:    nil,
	}
	res, err := RunNeoVMWithMap(bc, nil)
	if err != nil {
		t.Errorf("Empty bytecode should not return error, got %v", err)
	}
	if res != nil {
		t.Errorf("Empty bytecode should return nil, got %v", res)
	}
}

func TestNeoExVM_Safety_LargeStack(t *testing.T) {
	// Deep recursion but with variables to prevent constant folding if any
	input := "a"
	for i := 0; i < 100; i++ {
		input = fmt.Sprintf("a + (%s)", input)
	}
	engine, err := NewEngineVMNeo(input)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	_, err = engine.Execute(map[string]any{"a": int64(1)})
	if err == nil {
		t.Error("Expected stack overflow error for depth 100, got nil")
	}
}

func TestNeoExVM_Safety_PoolConcurrency(t *testing.T) {
	// Hammer the compiler pool to ensure no state leakage
	var wg sync.WaitGroup
	n := 1000
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			input := fmt.Sprintf("a + %d", idx)
			engine, err := NewEngineVMNeo(input)
			if err != nil {
				t.Errorf("Loop %d: Compile failed: %v", idx, err)
				return
			}
			res, err := engine.Execute(map[string]any{"a": int64(100)})
			if err != nil {
				t.Errorf("Loop %d: Execute failed: %v", idx, err)
				return
			}
			expected := int64(100 + idx)
			if res != expected {
				t.Errorf("Loop %d: Expected %v, got %v", idx, expected, res)
			}
		}(i)
	}
	wg.Wait()
}

func TestNeoExVM_Safety_ConstantTypes(t *testing.T) {
	// Test all supported constant types to ensure unsafe.Pointer access doesn't crash
	input := `if true is "string" else if false is 123 else if nil is 1.23 else is true`
	engine, err := NewEngineVMNeo(input)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	res, _ := engine.Execute(nil)
	if res != "string" {
		t.Errorf("Expected string, got %v", res)
	}
}

func TestNeoExVM_Safety_InvalidInput(t *testing.T) {
	inputs := []string{
		"if a then",
		"a + ",
		"(a + b",
		"if a is b else",
		"a = 1 +",
	}
	for _, input := range inputs {
		_, err := NewEngineVMNeo(input)
		if err == nil {
			t.Errorf("Expected error for invalid input %q, got nil", input)
		}
	}
}

func TestNeoExVM_Safety_CorruptBytecode(t *testing.T) {
	// PUSH with arg out of range
	bc := &NeoBytecode{
		Instructions: []neoInstruction{
			{Op: NeoOpPush, Arg: 999}, // Out of range
			{Op: NeoOpReturn, Arg: 0},
		},
		Constants: []Value{{Type: ValInt, Num: 1}},
	}

	if err := bc.Validate(); err == nil {
		t.Error("Expected validation error for out-of-range constant, got nil")
	}

	bc2 := &NeoBytecode{
		Instructions: []neoInstruction{
			{Op: NeoOpJump, Arg: 999}, // Out of range
		},
		Constants: []Value{{Type: ValInt, Num: 1}},
	}
	if err := bc2.Validate(); err == nil {
		t.Error("Expected validation error for out-of-range jump, got nil")
	}
}

func TestNeoExVM_Safety_Race(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race test in short mode")
	}

	input := `if a > 10 then b = a + 1 else b = a - 1`
	engine, _ := NewEngineVMNeo(input)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vars := map[string]any{"a": int64(20)}
			engine.Execute(vars)
		}()
	}
	wg.Wait()
}

func TestNeoExVM_Safety_GC(t *testing.T) {
	// Ensure bytecode remains valid after multiple GC cycles
	input := `if a == "test" then "ok" else "bad"`
	engine, _ := NewEngineVMNeo(input)

	for i := 0; i < 5; i++ {
		runtime.GC()
		res, _ := engine.Execute(map[string]any{"a": "test"})
		if res != "ok" {
			t.Errorf("GC cycle %d: Expected ok, got %v", i, res)
		}
	}
}
