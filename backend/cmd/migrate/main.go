package main

import (
	"context"
	"log"
	"os"
	"time"

	"veurubro/backend/internal/config"
	"veurubro/backend/internal/storage"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	databaseURL, err := config.RequiredSecret("DATABASE_URL")
	if err != nil {
		log.Fatal(err)
	}
	db, err := storage.Open(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if len(os.Args) > 1 && os.Args[1] == "down" {
		if err := db.MigrateDown(ctx); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := db.MigrateUp(ctx); err != nil {
		log.Fatal(err)
	}
	if err := db.SyncCatalog(ctx); err != nil {
		log.Fatal(err)
	}
}
