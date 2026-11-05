package main

import "fmt"

func main() {
    fmt.Println("Starting Heal CLI demonstration...")

    var message *string

    // Check if message is nil before dereferencing
    if message != nil {
        fmt.Println("Message:", *message)
    } else {
        fmt.Println("Message is nil, handling gracefully.")
    }

    fmt.Println("Application completed successfully!")
}