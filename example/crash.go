package main

import (
	"fmt"
	"log"
)

type Database interface {
	Connect() error
}

type PostgresDB struct {
	URL string
}

func (p *PostgresDB) Connect() error {
	// Simulate a connection failure
	return fmt.Errorf("connection refused: tcp timeout")
}

func main() {
	fmt.Println("Initializing services...")
	var db Database = &PostgresDB{URL: "localhost:5432"}
	if err := db.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
}