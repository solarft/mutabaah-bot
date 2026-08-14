package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"

	"github.com/solarft/mutabaah-bot/internal/appwrite"
	"github.com/solarft/mutabaah-bot/internal/config"
	"github.com/solarft/mutabaah-bot/internal/handlers"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	config.Init()

	pref := tele.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	appwrite.Init()
	handlers.HandleButtons(b)

	b.Start()
}
