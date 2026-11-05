package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/fatih/color"
)

func main() {
	// Intercept the re-exec call for the Linux namespace sandbox
	if len(os.Args) > 1 && os.Args[1] == "__child" {
		ChildSandboxEntrypoint()
		return
	}

	if len(os.Args) < 2 {
		color.Red("Usage: heal <command> [args...]")
		os.Exit(1)
	}

	llmClient, err := NewLLMClient()
	if err != nil {
		color.Red("[HEAL] Initialization Error: %v", err)
		os.Exit(1)
	}

	engine := NewRepairEngine()
	targetCmd := os.Args[1]
	targetArgs := os.Args[2:]
	maxRetries := 2

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			color.Cyan("\n[HEAL] --- Initiating Auto-Repair Attempt %d/%d ---", attempt, maxRetries)
		}

		exitCode, logString := executeWithPTY(targetCmd, targetArgs)
		if exitCode == 0 {
			msg := "Process completed successfully."
			if attempt > 0 {
				msg = "Auto-repair loop resolved all issues!"
			}
			color.Green("\n[HEAL] %s", msg)
			os.Exit(0)
		}

		color.Yellow("\n[HEAL] Process exited with error: exit status %v", exitCode)

		hash, isLooping := engine.FingerprintError(logString)
		if isLooping {
			color.Red("[HEAL] FATAL: Infinite loop detected. Hash %s seen before.", hash[:8])
			os.Exit(1)
		}

		color.Cyan("[HEAL] Error fingerprint: %s", hash[:8])

		crashSites := ExtractCrashContext(logString)
		if len(crashSites) == 0 {
			color.Yellow("[HEAL] No specific files/lines detected. Cannot generate patch.")
			os.Exit(1)
		}

		crashTarget := crashSites[0]
		filePath := strings.Split(crashTarget, ":")[0]

		color.Cyan("[HEAL] Extracting source context for %s...", crashTarget)
		sourceContext := ExtractContextWithLSP(crashTarget) // Updated to use LSP context engine

		color.Cyan("[HEAL] Expanding state space. Requesting 3 variants from LLM...")
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		patches, err := llmClient.RequestPatchCandidates(ctx, logString, sourceContext, 3)
		cancel()

		if err != nil {
			color.Red("[HEAL] LLM Request failed: %v", err)
			os.Exit(1)
		}

		bestNode, searchErr := Explore(
			filePath,
			append([]string{targetCmd}, targetArgs...),
			patches,
		)
		if searchErr != nil {
			color.Red("[HEAL] Search tree collapsed: %v", searchErr)
			continue // Trigger the next attempt loop
		}

		// Apply the winning node's patch to the host
		if bestNode.State == StatePerfect || bestNode.State == StateGracefulDegradation {
			if applyErr := os.WriteFile(filePath, []byte(bestNode.Patch), 0644); applyErr != nil {
				color.Red("[HEAL] Failed to write optimal patch to host: %v", applyErr)
				os.Exit(1)
			}
			color.Green("\n[HEAL] MCTS Repair successful. File %s updated.", filePath)

			color.Cyan("\n[HEAL] Re-running application to verify repair...")

			verifyExitCode, _ := executeWithPTY(targetCmd, targetArgs)

			if verifyExitCode == 0 {
				color.Green("\n[HEAL] Verification successful! Application is healthy.")
				os.Exit(0)
			}

			color.Red("\n[HEAL] Verification failed. Application still has errors.")
			os.Exit(1)
		}
	}

	color.Red("\n[HEAL] Exhausted retry budget (%d attempts). Repair failed.", maxRetries)
	os.Exit(1)
}

// executeWithPTY encapsulates the terminal multiplexing logic
// executeWithPTY encapsulates the terminal multiplexing logic
func executeWithPTY(command string, args []string) (int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return 1, fmt.Sprintf("Error starting pty: %v", err)
	}
	defer ptmx.Close()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	ch <- syscall.SIGWINCH

	var outBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &outBuffer)

	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	go func() { _, _ = io.Copy(mw, ptmx) }()

	err = cmd.Wait()
	signal.Stop(ch)
	close(ch)

	// Intercept the timeout and inject a synthetic stack trace
	if ctx.Err() == context.DeadlineExceeded {
		var targetFile string
		for _, arg := range args {
			if strings.HasSuffix(arg, ".go") {
				targetFile = arg
				break
			}
		}
		if targetFile == "" {
			targetFile = "main.go" // Fallback
		}

		// Spoof a standard Go panic trace so the context extractor can find the file
		timeoutMsg := fmt.Sprintf("\npanic: process timed out (infinite loop or deadlock detected)\n\ngoroutine 1 [running]:\nmain.main()\n\t%s:1 +0x0\n", targetFile)

		fmt.Fprintln(os.Stdout, color.RedString(timeoutMsg))
		outBuffer.WriteString(timeoutMsg)
		return 124, outBuffer.String()
	}

	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	return exitCode, outBuffer.String()
}
