package main
import (
    "log"
    "os"
)

func main() {
    // Simulates an AI hallucinating a destructive command
    err := os.RemoveAll("/tmp/test_host_wipe")
    if err != nil {
        log.Printf("Error removing directory: %v", err)
        return
    }
    log.Println("Directory removed successfully")
}