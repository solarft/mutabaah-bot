package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/solarft/mutabaah-bot/internal/appwrite"
	tele "gopkg.in/telebot.v4"
)

func HandleButtons(b *tele.Bot) {
	var (
		// Universal markup builders.
		menu     = &tele.ReplyMarkup{ResizeKeyboard: true}
		selector = &tele.ReplyMarkup{}

		// Reply buttons.
		btnHelp       = menu.Text("ℹ Help")
		btnSettings   = menu.Text("⚙ Settings")
		btnStats      = menu.Text("Stats")
		btnListSunnah = menu.Text("List Sunnah Logs")

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
		menu.Row(btnHelp, btnSettings),
		menu.Row(btnStats, btnListSunnah),
	)
	selector.Inline(
		selector.Row(btnPrev, btnNext),
	)

	b.Handle("/start", func(c tele.Context) error {
		username := c.Sender().Username
		if username == "" {
			return c.Send("Please set a Telegram username in your account settings, then try again.", menu)
		}

		if err := appwrite.SetTelegramID(username, c.Sender().ID); err != nil {
			return c.Send("Error: " + err.Error())
		}

		data, err := appwrite.GetData(c.Sender().ID)
		if err != nil {
			if errors.Is(err, appwrite.ErrNotFound) {
				return c.Send("Your username was not found. Please put your username in the website to finish account creation.", menu)
			}
			return c.Send("Error: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Hello @%s!\nYour data: %v", username, data), menu)
	})

	// On reply button pressed (message)
	b.Handle(&btnHelp, func(c tele.Context) error {
		return c.Edit("Here is some help: ...")
	})

	b.Handle(&btnStats, func(c tele.Context) error {
		data, err := appwrite.GetData(c.Sender().ID)
		if err != nil {
			if errors.Is(err, appwrite.ErrNotFound) {
				return c.Send("Your username was not found. Please put your username in the website to finish account creation.", menu)
			}
			return c.Send("Error: " + err.Error())
		}

		return c.Send(fmt.Sprintf("data:\n %v", data))
	})

	b.Handle(&btnListSunnah, func(c tele.Context) error {
		logs, err := appwrite.ListSunnahLogs(c.Sender().ID)
		if err != nil {
			return c.Send("Error: " + err.Error())
		}
		if len(logs) == 0 {
			return c.Send("No sunnah logs found.", menu)
		}

		var sb strings.Builder
		for _, log := range logs {
			fmt.Fprintf(&sb, "%s\n", log.Date)
			for i, item := range log.Items {
				fmt.Fprintf(&sb, "	%d. %s\n", i+1, item)
			}
		}
		return c.Send(sb.String(), menu)
	})

	// On inline button pressed (callback)
	b.Handle(&btnPrev, func(c tele.Context) error {
		return c.Respond()
	})
}
