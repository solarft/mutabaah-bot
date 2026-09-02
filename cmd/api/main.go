package main

import (
	"log"
	"os"
	"time"
	_ "time/tzdata"

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

	if os.Getenv("WEBHOOK_MODE") == "1" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		log.Println("Using webhook mode")
		pref.Poller = &tele.Webhook{
			Listen:           ":" + port,
			Endpoint:         &tele.WebhookEndpoint{PublicURL: os.Getenv("WEBHOOK_URL")},
			SecretToken:      os.Getenv("WEBHOOK_SECRET"),
			IgnoreSetWebhook: true,
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
	handlers.HandleButtons(b)

	go scheduleDaily(appwrite.LogMurajaah)

	// Start the bot in a goroutine so the HTTP server begins listening
	// immediately. This is critical on Cloud Run / Vercel where the
	// platform must see the server listening on PORT within the startup
	// timeout, otherwise it returns 502.
	go b.Start()

	// Set the webhook with Telegram. With IgnoreSetWebhook=true above,
	// Poll() won't call SetWebhook again, so the HTTP server was
	// already listening when this runs.
	if webhook, ok := pref.Poller.(*tele.Webhook); ok {
		if err := b.SetWebhook(webhook); err != nil {
			log.Printf("Warning: SetWebhook failed: %v", err)
		}
	}

	log.Printf("Env check: APPWRITE_ENDPOINT=%v, APPWRITE_PROJECT_ID=%v, APPWRITE_KEY=%v, DATABASE_ID=%v, USERS_TABLE_ID=%v, SUNNAH_LOGS_TABLE_ID=%v",
		os.Getenv("APPWRITE_ENDPOINT") != "",
		os.Getenv("APPWRITE_PROJECT_ID") != "",
		os.Getenv("APPWRITE_KEY") != "",
		os.Getenv("DATABASE_ID") != "",
		os.Getenv("USERS_TABLE_ID") != "",
		os.Getenv("SUNNAH_LOGS_TABLE_ID") != "",
	)

	log.Println("Bot started")
	select {} // block forever
}

func scheduleDaily(f func() error) {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			time.Sleep(next.Sub(now))
			if err := f(); err != nil {
				log.Printf("daily task error: %v", err)
			}
		}
	}()
}
