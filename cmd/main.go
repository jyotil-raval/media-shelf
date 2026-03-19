// cmd/main.go
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/jyotil-raval/media-shelf/internal/db"
)

func main() {
	godotenv.Load() // optional — .env present locally, injected by Docker in container

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	database, err := db.Open(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("media-shelf ready.")
}
