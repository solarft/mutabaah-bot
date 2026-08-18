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

	if os.Getenv("VERCEL") != "" {
		port := os.Getenv("PORT")
		log.Println("Using webhook mode")
		//	pref.Synchronous = true
		pref.Poller = &tele.Webhook{
			Listen:      ":" + port,
			Endpoint:    &tele.WebhookEndpoint{PublicURL: "https://" + os.Getenv("VERCEL_URL") + "/webhook"},
			SecretToken: os.Getenv("WEBHOOK_SECRET"),
		}
	} else {
		log.Println("Using long polling mode")
		pref.Poller = &tele.LongPoller{Timeout: 10 * time.Second}
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	appwrite.Init()
	if os.Getenv("VERCEL") != "" {
		log.Println("Pre-warming Appwrite connection")
		appwrite.Prewarm()
	}
	handlers.HandleButtons(b)
	log.Println("Bot started")
	b.Start()
}
