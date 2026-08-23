package handlers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/solarft/mutabaah-bot/internal/appwrite"
	tele "gopkg.in/telebot.v4"
)

var amalan = []string{
	"Qiamullail",
	"Ma'thurat",
	"Dhuha Prayer",
	"Solat Berjemaah",
	"Solat Sunat Rawatib",
	"Read 1 Juz of Quran",
	"Murajaah Quran",
	"Istighfar 100×",
	"Selawat 100×",
	"Muhasabah Diri",
}

var userSelections = make(map[int64][]string)

func buildSunnahChecklist(selected []string) (string, *tele.ReplyMarkup) {
	completed := make(map[string]bool)
	for _, item := range selected {
		completed[item] = true
	}

	var sb strings.Builder
	sb.WriteString("Please select which deeds you have done:\nDate: ")
	date := time.Now().Format(time.DateOnly)
	sb.WriteString(date)
	sb.WriteString("\n\n")

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, name := range amalan {
		done := completed[name]
		statusEmoji := "❌ "
		if done {
			statusEmoji = "✅ "
		}

		sb.WriteString(statusEmoji)
		sb.WriteString(name)
		sb.WriteString("\n")

		btn := markup.Data(statusEmoji+name, "toggle_sunnah", name)
		rows = append(rows, markup.Row(btn))
	}

	submitBtn := markup.Data("Submit", "submit_sunnah")
	rows = append(rows, markup.Row(submitBtn))

	markup.Inline(rows...)
	return sb.String(), markup
}

func buildChecklistText(selected []string) string {
	completed := make(map[string]bool)
	for _, item := range selected {
		completed[item] = true
	}

	var sb strings.Builder
	sb.WriteString("Please select which deeds you have done:\nDate: ")
	date := time.Now().Format(time.DateOnly)
	sb.WriteString(date)
	sb.WriteString("\n\n")
	for _, name := range amalan {
		statusEmoji := "❌ "
		if completed[name] {
			statusEmoji = "✅ "
		}
		sb.WriteString(statusEmoji)
		sb.WriteString(name)
		sb.WriteString("\n")
	}
	return sb.String()
}

func HandleButtons(b *tele.Bot) {
	var (
		// Universal markup builders.
		menu           = &tele.ReplyMarkup{ResizeKeyboard: true}
		selector       = &tele.ReplyMarkup{}
		sunnahSelector = &tele.ReplyMarkup{}

		// Reply buttons.
		btnHelp       = menu.Text("ℹ Help")
		btnSettings   = menu.Text("⚙ Settings")
		btnStats      = menu.Text("Stats")
		btnListSunnah = menu.Text("List Sunnah Logs")
		btnAddSunnah  = menu.Text("Add Sunnah")

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
		menu.Row(btnAddSunnah),
		menu.Row(btnHelp, btnSettings),
		menu.Row(btnStats, btnListSunnah),
	)
	selector.Inline(
		selector.Row(btnPrev, btnNext),
	)

	sunnahBtnHandler := sunnahSelector.Data("", "toggle_sunnah")

	b.Handle("/ping", func(c tele.Context) error {
		return c.Send("Pong!")
	})

	b.Handle("/debug", func(c tele.Context) error {
		var sb strings.Builder

		fmt.Fprintf(&sb, "Appwrite endpoint: %s\n", os.Getenv("APPWRITE_ENDPOINT"))

		lookup := func(name, host string) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			ips, err := net.DefaultResolver.LookupHost(ctx, host)
			if err != nil {
				fmt.Fprintf(&sb, "  %s lookup: ERROR %v\n", name, err)
			} else {
				fmt.Fprintf(&sb, "  %s lookup: OK %v\n", name, ips)
			}
		}

		fmt.Fprintf(&sb, "Default resolver:\n")
		lookup("appwrite.network", "appwrite.network")
		lookup("api.telegram.org", "api.telegram.org")

		alt := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, network, "1.1.1.1:53")
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		ips, err := alt.LookupHost(ctx, "appwrite.network")
		cancel()
		if err != nil {
			fmt.Fprintf(&sb, "Resolver via 1.1.1.1:53:\n  appwrite.network: ERROR %v\n", err)
		} else {
			fmt.Fprintf(&sb, "Resolver via 1.1.1.1:53:\n  appwrite.network: OK %v\n", ips)
		}

		conn, err := net.DialTimeout("tcp", "151.101.3.52:443", 5*time.Second)
		if err != nil {
			fmt.Fprintf(&sb, "dial 151.101.3.52:443: ERROR %v\n", err)
		} else {
			conn.Close()
			fmt.Fprintf(&sb, "dial 151.101.3.52:443: OK\n")
		}

		client := &http.Client{Timeout: 10 * time.Second}
		for _, u := range []string{"https://appwrite.network", "https://api.telegram.org"} {
			resp, err := client.Get(u)
			if err != nil {
				fmt.Fprintf(&sb, "GET %s: ERROR %v\n", u, err)
			} else {
				fmt.Fprintf(&sb, "GET %s: %d\n", u, resp.StatusCode)
				resp.Body.Close()
			}
		}

		return c.Send(sb.String())
	})

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
		return c.Send("Here is some help: ...", menu)
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

	b.Handle(&btnAddSunnah, func(c tele.Context) error {
		logs, err := appwrite.ListSunnahLogs(c.Sender().ID)
		if err != nil {
			fmt.Printf("Failed to get sunnah log: %v\n", err)
		}

		var current []string
		if len(logs) > 0 {
			current = logs[len(logs)-1].Items
		}
		userSelections[c.Sender().ID] = current

		text, markup := buildSunnahChecklist(current)
		return c.Send(text, markup)
	})

	b.Handle(&sunnahBtnHandler, func(c tele.Context) error {
		sel := userSelections[c.Sender().ID]
		activity := c.Data()

		found := -1
		for i, item := range sel {
			if item == activity {
				found = i
				break
			}
		}
		if found >= 0 {
			sel = append(sel[:found], sel[found+1:]...)
		} else {
			sel = append(sel, activity)
		}
		userSelections[c.Sender().ID] = sel

		text, markup := buildSunnahChecklist(sel)
		return c.Edit(text, markup)
	})

	submitBtnHandler := sunnahSelector.Data("Submit", "submit_sunnah")
	b.Handle(&submitBtnHandler, func(c tele.Context) error {
		sel := userSelections[c.Sender().ID]
		delete(userSelections, c.Sender().ID)

		if _, err := appwrite.SaveSunnahSelections(c.Sender().ID, sel); err != nil {
			return c.Send("Error saving: " + err.Error())
		}

		return c.Edit(buildChecklistText(sel))
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
