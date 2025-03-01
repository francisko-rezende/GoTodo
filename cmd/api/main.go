package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
		os.Exit(1)
	}

	dsn := os.Getenv("DB_DSN")

	fmt.Println("Hello, GoTodo")
}
