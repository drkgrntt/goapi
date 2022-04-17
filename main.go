package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
	
	app := fiber.New()
	db := initDb()

	initUserRoutes(app, db)
	initAuthRoutes(app, db)

	port := os.Getenv("PORT")
	
	log.Fatalln(app.Listen(fmt.Sprintf(":%v", port)))
}
