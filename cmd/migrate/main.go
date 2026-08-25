package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kingmon6996/orbit/internal/config"
	"github.com/kingmon6996/orbit/internal/database"
	"github.com/kingmon6996/orbit/internal/logging"
)

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		log.Print("usage: go run ./cmd/migrate [up|down]")
		return
	}
	configuration, err := config.Load()
	if err != nil {
		log.Printf("failed to load configuration: %v", err)
		return
	}
	if !configuration.DatabaseEnabled {
		log.Print("DATABASE_ENABLED=true is required to run migrations")
		return
	}
	logger := logging.New(configuration)
	databaseConnection, err := database.New(context.Background(), configuration, logger)
	if err != nil {
		log.Printf("failed to connect to database: %v", err)
		return
	}
	defer databaseConnection.Close()
	if os.Args[1] == "up" {
		err = database.MigrateUp(context.Background(), databaseConnection.Pool())
	} else {
		err = database.MigrateDown(context.Background(), databaseConnection.Pool())
	}
	if err != nil {
		log.Printf("migration %s failed: %v", os.Args[1], err)
		return
	}
	fmt.Printf("migration %s completed\n", os.Args[1])
}
