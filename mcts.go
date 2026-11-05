package main

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
)

type SearchNode struct {
	Patch string
	State ExecutionState
	Score int
}

// Map execution states to formal heuristic rewards
func getReward(state ExecutionState) int {
	switch state {
	case StatePerfect:
		return 100
	case StateGracefulDegradation:
		return 50
	case StateCompilerError:
		return -10
	case StateUnhandledPanic:
		return -50
	default:
		return 0
	}
}

// Explore concurrently simulates N patch branches and returns the highest scoring node
func Explore(filePath string, targetArgs []string, patches []string) (*SearchNode, error) {
	color.Magenta("\n[MCTS] Expanding search tree: Simulating %d candidate patches concurrently...", len(patches))

	results := make(chan *SearchNode, len(patches))
	var wg sync.WaitGroup

	for i, patch := range patches {
		wg.Add(1)
		go func(nodeID int, candidate string) {
			defer wg.Done()
			
			// Pass a flag to RunInSandbox so it doesn't overwrite the host file during simulation
			state, err := SimulateInSandbox(filePath, candidate, targetArgs)
			if err != nil {
				results <- &SearchNode{Patch: candidate, State: StateCompilerError, Score: -100}
				return
			}
			
			score := getReward(state)
			color.Cyan("[MCTS-SIM] Node %d evaluated: %s (Score: %d)", nodeID, state, score)
			results <- &SearchNode{Patch: candidate, State: state, Score: score}
		}(i, patch)
	}

	wg.Wait()
	close(results)

	var bestNode *SearchNode
	highestScore := -999

	for node := range results {
		if node.Score > highestScore {
			highestScore = node.Score
			bestNode = node
		}
	}

	if bestNode == nil || bestNode.Score <= 0 {
		return nil, fmt.Errorf("all branches collapsed into fatal errors")
	}

	color.Green("\n[MCTS] Backpropagation complete. Selected optimal patch branch with score: %d", bestNode.Score)
	return bestNode, nil
}