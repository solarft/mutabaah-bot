package handlers

import (
	"github.com/appwrite/sdk-for-go/id"
	"github.com/solarft/mutabaah-bot/internal/appwrite"
	"github.com/solarft/mutabaah-bot/internal/config"
	tele "gopkg.in/telebot.v4"
)

func HandleButtons(b *tele.Bot) {
	var (
		// Universal markup builders.
		menu     = &tele.ReplyMarkup{ResizeKeyboard: true}
		selector = &tele.ReplyMarkup{}

		// Reply buttons.
		btnHelp     = menu.Text("ℹ Help")
		btnSettings = menu.Text("⚙ Settingscompiler: undefined: menu.MenuInit")

		// Inline buttons.
		//
		// Pressing it will cause the client to
		// send the bot a callback.
		//
		// Make sure Unique stays unique as per button kind
		// since it's required for callback routing to work.
		//
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
		_, err := appwrite.TablesDB().CreateRow(
			config.DatabaseID,
			config.UsersTableID,
			id.Unique(),
			map[string]interface{}{
				"data":              "{}",
				"telegram_id":       c.Sender().ID,
				"telegram_username": c.Sender().Username,
			})
		if err != nil {
			return c.Send("Error: " + err.Error())
		}
		response := "Hello " + c.Sender().Username + "! You can go back to the website to finish account creation."
		return c.Send(response, menu)
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
