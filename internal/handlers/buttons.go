package handlers

import (
	"errors"
	"fmt"

	"github.com/solarft/mutabaah-bot/internal/appwrite"
	tele "gopkg.in/telebot.v4"
)

func HandleButtons(b *tele.Bot) {
	var (
		// Universal markup builders.
		menu     = &tele.ReplyMarkup{ResizeKeyboard: true}
		selector = &tele.ReplyMarkup{}

		// Reply buttons.
		btnHelp     = menu.Text("ℹ Help")
		btnSettings = menu.Text("⚙ Settings")

		// Inline buttons.
		//
		// Pressing it will cause the client to
		// send the bot a callback.
		//
		// Make sure Unique stays unique as per button kind
		// since it's required for callback routing to work.
		// ^^ This is just copy-pasted hence the comments. i'll work on the buttons later
		btnPrev = selector.Data("⬅", "prev")
		btnNext = selector.Data("➡", "next")
	)

	menu.Reply(
		menu.Row(btnHelp),
		menu.Row(btnSettings),
	)
	selector.Inline(
		selector.Row(btnPrev, btnNext),
	)

	b.Handle("/start", func(c tele.Context) error {
		username := c.Sender().Username
		if username == "" {
			return c.Send("Please set a Telegram username in your account settings, then try again.", menu)
		}

		data, err := appwrite.GetData(username)
		if err != nil {
			if errors.Is(err, appwrite.ErrNotFound) {
				return c.Send("Your username was not found. Please put your username in the website to finish account creation.", menu)
			}
			return c.Send("Error: " + err.Error())
		}

		if err := appwrite.SetTelegramID(username, c.Sender().ID); err != nil {
			return c.Send("Error: " + err.Error())
		}
		return c.Send(fmt.Sprintf("Hello @%s!\nYour data: %v", username, data), menu)
	})

	// On reply button pressed (message)
	b.Handle(&btnHelp, func(c tele.Context) error {
		return c.Edit("Here is some help: ...")
	})

	// On inline button pressed (callback)
	b.Handle(&btnPrev, func(c tele.Context) error {
		return c.Respond()
	})
}
