package main

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// RepairEngine encapsulates state to avoid global variables
type RepairEngine struct {
	stateHistory map[string]bool
}

func NewRepairEngine() *RepairEngine {
	return &RepairEngine{
		stateHistory: make(map[string]bool),
	}
}

// FingerprintError checks for infinite loops using cryptographic hashing
func (e *RepairEngine) FingerprintError(logOutput string) (string, bool) {
	hash := sha256.Sum256([]byte(strings.TrimSpace(logOutput)))
	hashStr := hex.EncodeToString(hash[:])

	isLooping := e.stateHistory[hashStr]
	e.stateHistory[hashStr] = true

	return hashStr, isLooping
}

// ExtractCrashContext remains a pure utility function
func ExtractCrashContext(logOutput string) []string {
	re := regexp.MustCompile(`([a-zA-Z0-9_\-\./\\]+\.[a-zA-Z]+):(\d+)`)
	matches := re.FindAllString(logOutput, -1)

	unique := make(map[string]bool)
	var files []string
	
	for _, match := range matches {
		if !unique[match] {
			unique[match] = true
			files = append(files, match)
		}
	}
	return files
}