package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/jyotil-raval/media-shelf/cmd/shelf"
	"github.com/jyotil-raval/media-shelf/internal/db"
	"github.com/jyotil-raval/media-shelf/internal/providers/mal"
)

func main() {
	godotenv.Load()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	grpcTarget := os.Getenv("MAL_UPDATER_GRPC_URL")
	if grpcTarget == "" {
		grpcTarget = "localhost:9090"
	}

	database, err := db.Open(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	malClient, err := mal.NewClient(grpcTarget)
	if err != nil {
		log.Fatalf("connecting to mal-updater: %v", err)
	}
	defer malClient.Close()

	store := db.NewPostgreSQLStore(database)
	app := shelf.NewApp(store, malClient)
	root := shelf.NewRootCommand(app)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
