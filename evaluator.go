package main

import (
	"strings"
)

// ExecutionState categorizes the architectural health of the process
type ExecutionState string

const (
	StatePerfect             ExecutionState = "Perfect Execution (Exit 0)"
	StateGracefulDegradation ExecutionState = "Graceful Degradation (Handled Error)"
	StateCompilerError       ExecutionState = "Compiler/Syntax Error"
	StateUnhandledPanic      ExecutionState = "Unhandled Panic"
)

// EvaluateState acts as the heuristic scoring function for our bounded search.
func EvaluateState(exitCode int, stdErr string) ExecutionState {
	if exitCode == 0 {
		return StatePerfect
	}

	// Did the LLM hallucinate invalid syntax?
	if strings.Contains(stdErr, "syntax error") || strings.Contains(stdErr, "undefined:") || strings.Contains(stdErr, "expected") {
		return StateCompilerError
	}

	// Did the process still blow up with a panic?
	if strings.Contains(stdErr, "panic:") {
		return StateUnhandledPanic
	}

	// If it exited non-zero but didn't panic and compiled successfully,
	// the LLM successfully contained the blast radius (e.g., log.Fatalf).
	return StateGracefulDegradation
} 