package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// SimulateInSandbox provisions an ephemeral workspace, forks a namespace, and scores the result.
func SimulateInSandbox(filePath, patch string, targetArgs []string) (ExecutionState, error) {
	sandboxDir, err := os.MkdirTemp("", "heal_sandbox_*")
	if err != nil {
		return StateCompilerError, fmt.Errorf("failed to create sandbox dir: %w", err)
	}
	defer os.RemoveAll(sandboxDir)

	baseName := filepath.Base(filePath)
	sandboxFilePath := filepath.Join(sandboxDir, baseName)

	if err := os.WriteFile(sandboxFilePath, []byte(patch), 0644); err != nil {
		return StateCompilerError, fmt.Errorf("failed to write patch to sandbox: %w", err)
	}

	sandboxArgs := make([]string, len(targetArgs))
	copy(sandboxArgs, targetArgs)
	for i, arg := range sandboxArgs {
		if stringsContains(arg, baseName) {
			sandboxArgs[i] = sandboxFilePath
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := append([]string{"__child", sandboxDir}, sandboxArgs...)
	cmd := exec.CommandContext(ctx, "/proc/self/exe", args...)

	// Retain rootless User isolation for the namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
		GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
	}

	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &outBuffer

	err = cmd.Run()

	// Intercept context timeouts before standard exit codes
	if ctx.Err() == context.DeadlineExceeded {
		return StateUnhandledPanic, fmt.Errorf("sandbox execution timed out (infinite loop/fork bomb detected)")
	}
	
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	state := EvaluateState(exitCode, outBuffer.String())
	return state, nil
}

// ChildSandboxEntrypoint is executed INSIDE the isolated kernel namespace.
func ChildSandboxEntrypoint() {
	if len(os.Args) < 4 {
		os.Exit(1)
	}

	sandboxDir := os.Args[2]
	targetCmdArgs := os.Args[3:]

	cmd := exec.Command(targetCmdArgs[0], targetCmdArgs[1:]...)
	cmd.Dir = sandboxDir // Force execution inside the sandbox directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// Helper to avoid importing "strings" just for one check
func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}