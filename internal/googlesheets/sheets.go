package googlesheets

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	SpreadsheetID string
	srv           *sheets.Service
)

func InitSheets() {
	ctx := context.Background()
	var err error
	srv, err = sheets.NewService(ctx, option.WithAuthCredentialsFile(option.ServiceAccount, "service-account.json"))
	if err != nil {
		log.Fatal(err)
	}
	SpreadsheetID = os.Getenv("SPREADSHEET_ID")
}

func ReadSheet(start string, end string) {
	readRange := "Class Data!A2:E"
	resp, err := srv.Spreadsheets.Values.Get(SpreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		fmt.Println("Name, Major:")
		for _, row := range resp.Values {
			// Print columns A and E, which correspond to indices 0 and 4.
			fmt.Printf("%s, %s\n", row[0], row[4])
		}
	}
}

func WriteSheet(r string, values *sheets.ValueRange) {
	_, err := srv.Spreadsheets.Values.Update(SpreadsheetID, r, values).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		log.Fatalf("Unable to write data to sheet: %v", err)
	}
	fmt.Println("Row updated.")
}

func AppendSheet(r string, values *sheets.ValueRange) {
	_, err := srv.Spreadsheets.Values.Append(SpreadsheetID, r, values).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		log.Fatalf("Unable to append data to sheet: %v", err)
	}
	fmt.Println("Row appended.")
}
