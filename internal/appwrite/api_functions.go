package appwrite

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/appwrite/sdk-for-go/query"
	"github.com/solarft/mutabaah-bot/internal/config"
)

var ErrNotFound = errors.New("no row found for this telegram username")

type Row struct {
	ID   string          `json:"$id"`
	Data json.RawMessage `json:"data"`
}

type Payload struct {
	Rows []Row `json:"rows"`
}

type SunnahLog struct {
	Date  string
	Items []string
}

// decodes json -> map[string]any
func parseData(raw json.RawMessage) (map[string]any, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err == nil {
		return data, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, errors.New("data is not a JSON object or string")
	}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return nil, errors.New("data string contains invalid JSON")
	}
	return data, nil
}

func ParseList(raw json.RawMessage) ([]string, error) {
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, errors.New("value is not a JSON array or string")
	}
	if err := json.Unmarshal([]byte(s), &list); err != nil {
		return nil, errors.New("array string contains invalid JSON")
	}
	return list, nil
}

// Prewarm opens a connection to Appwrite at startup so the transport
// pool reuses it instead of opening new connections mid-request.
func Prewarm() {
	_, err := tablesDB.ListRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithListRowsQueries([]string{query.Equal("telegram_id", 0)}),
	)
	if err != nil {
		log.Printf("appwrite prewarm: %v", err)
	}
}

// GetData returns the data column for the row matching telegram_username.
func GetData(telegramID int64) (map[string]any, error) {
	rows, err := tablesDB.ListRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithListRowsQueries([]string{
			query.Equal("telegram_id", telegramID),
		}),
	)
	if err != nil {
		return nil, err
	}

	var payload Payload

	if err := rows.Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Rows) == 0 {
		return nil, ErrNotFound
	}
	return parseData(payload.Rows[0].Data)
}

func ListSunnahLogs(telegramID int64) ([]SunnahLog, error) {
	rows, err := tablesDB.ListRows(
		config.DatabaseID, config.SunnahLogsTableID,
		tablesDB.WithListRowsQueries([]string{
			query.Equal("telegram_id", telegramID),
			query.OrderAsc("date"),
		}),
	)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Rows []struct {
			Date string          `json:"date"`
			Data json.RawMessage `json:"data"`
		} `json:"rows"`
	}
	if err := rows.Decode(&payload); err != nil {
		return nil, err
	}

	logs := make([]SunnahLog, 0, len(payload.Rows))
	for _, row := range payload.Rows {
		items, err := ParseList(row.Data)
		if err != nil {
			return nil, err
		}

		logs = append(logs, SunnahLog{Date: row.Date, Items: items})
	}
	return logs, nil
}

// SetData updates the data column on all rows matching telegram_username.
func SetData(telegramID string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tablesDB.UpdateRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithUpdateRowsData(map[string]any{"data": string(raw)}),
		tablesDB.WithUpdateRowsQueries([]string{
			query.Equal("telegram_id", telegramID),
		}),
	)
	return err
}

// SetTelegramID updates the telegram_id column on all rows matching telegram_username.
func SetTelegramID(username string, telegramID int64) error {
	_, err := tablesDB.UpdateRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithUpdateRowsData(map[string]any{"telegram_id": telegramID}),
		tablesDB.WithUpdateRowsQueries([]string{
			query.Equal("telegram_username", username),
		}),
	)
	return err
}
