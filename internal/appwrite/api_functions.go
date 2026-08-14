package appwrite

import (
	"encoding/json"
	"errors"

	"github.com/appwrite/sdk-for-go/query"
	"github.com/solarft/mutabaah-bot/internal/config"
)

var ErrNotFound = errors.New("no row found for this telegram username")

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

// GetData returns the data column for the row matching telegram_username.
func GetData(username string) (map[string]any, error) {
	rows, err := tablesDB.ListRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithListRowsQueries([]string{
			query.Equal("telegram_username", username),
		}),
	)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Rows []struct {
			ID   string          `json:"$id"`
			Data json.RawMessage `json:"data"`
		} `json:"rows"`
	}
	if err := rows.Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Rows) == 0 {
		return nil, ErrNotFound
	}
	return parseData(payload.Rows[0].Data)
}

// SetData updates the data column on all rows matching telegram_username.
func SetData(username string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tablesDB.UpdateRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithUpdateRowsData(map[string]any{"data": string(raw)}),
		tablesDB.WithUpdateRowsQueries([]string{
			query.Equal("telegram_username", username),
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
