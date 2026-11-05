package main

import (
    "log"
    "time"
)

func main() {
    for i := 0; i < 10; i++ { // Limit the number of goroutines
        go func() {
            for {
                time.Sleep(1 * time.Second) // Prevents infinite loop from consuming all CPU
            }
        }()
    }

    // Prevent main from exiting immediately
    select {}
}