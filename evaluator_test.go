package main

import "testing"

func TestEvaluateState(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		stdErr   string
		expected ExecutionState
	}{
		{"Perfect Exit", 0, "", StatePerfect},
		{"Syntax Error", 1, "undefined: fmt", StateCompilerError},
		{"Unhandled Panic", 2, "panic: runtime error: invalid memory address", StateUnhandledPanic},
		{"Graceful Degradation", 1, "2026/11/05 Failed to connect", StateGracefulDegradation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateState(tt.exitCode, tt.stdErr)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}