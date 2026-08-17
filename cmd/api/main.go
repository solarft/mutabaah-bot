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
	_ = godotenv.Load()
	config.Init()

	pref := tele.Settings{
		Token: os.Getenv("TOKEN"),
	}

	if port := os.Getenv("PORT"); port != "" {
		pref.Synchronous = true
		pref.Poller = &tele.Webhook{
			Listen:      ":" + port,
			Endpoint:    &tele.WebhookEndpoint{PublicURL: "https://" + os.Getenv("VERCEL_URL") + "/webhook"},
			SecretToken: os.Getenv("WEBHOOK_SECRET"),
		}
	} else {
		pref.Poller = &tele.LongPoller{Timeout: 10 * time.Second}
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	appwrite.Init()
	handlers.HandleButtons(b)
	b.Start()
}
