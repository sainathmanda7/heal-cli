package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// ExtractContextWithLSP queries the Go Language Server for semantic workspace context
func ExtractContextWithLSP(crashSite string) string {
	parts := strings.Split(crashSite, ":")
	if len(parts) != 2 {
		return ""
	}
	filePath := parts[0]
	lineNum, _ := strconv.Atoi(parts[1])

	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("CRASH SITE: %s\n\n", crashSite))

	// 1. Resolve gopls path gracefully
	goplsBin := "gopls"
	if _, err := exec.LookPath(goplsBin); err != nil {
		if home, err := os.UserHomeDir(); err == nil {
			goplsBin = filepath.Join(home, "go", "bin", "gopls")
		}
	}

	// 2. Query LSP for Semantic Symbols (The File Skeleton)
	color.Cyan("[HEAL-LSP] Querying LSP for semantic workspace symbols...")
	symbolCmd := exec.Command(goplsBin, "symbols", filePath)
	var symOut bytes.Buffer
	symbolCmd.Stdout = &symOut
	
	if err := symbolCmd.Run(); err == nil && symOut.Len() > 0 {
		contextBuilder.WriteString("--- FILE ARCHITECTURE (LSP SYMBOLS) ---\n")
		contextBuilder.WriteString(symOut.String())
		contextBuilder.WriteString("\n---------------------------------------\n\n")
	} else {
		color.Yellow("[HEAL-LSP] Warning: gopls not found or workspace uninitialized.")
	}

	// 3. Extract the local crash window (The Symptom)
	contextBuilder.WriteString("--- LOCAL CRASH WINDOW ---\n")
	file, err := os.Open(filePath)
	if err != nil {
		return contextBuilder.String()
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	start := lineNum - 12
	if start < 0 { start = 0 }
	end := lineNum + 12
	if end > len(lines) { end = len(lines) }

	for i := start; i < end; i++ {
		marker := "  "
		if i+1 == lineNum {
			marker = ">>" 
		}
		contextBuilder.WriteString(fmt.Sprintf("%s %d: %s\n", marker, i+1, lines[i]))
	}
	contextBuilder.WriteString("--------------------------\n")

	return contextBuilder.String()
}