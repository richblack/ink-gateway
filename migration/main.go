package main

import (
	"log"
	"os"

	"semantic-text-processor/migration"
)

func main() {
	cli := migration.NewMigrationCLI()

	if err := cli.Run(); err != nil {
		log.Printf("Migration CLI error: %v", err)
		os.Exit(1)
	}
}