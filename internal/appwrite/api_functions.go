package appwrite

import (
	"encoding/json"
	"errors"
	"log"
	"time"

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

type MurajaahPayload struct {
	PageRatings     json.RawMessage `json:"murajaahPageRatings"`
	TasmikRatings   json.RawMessage `json:"murajaahTasmikRatings"`
	RepetitionTicks json.RawMessage `json:"murajaahRepetitionTicks"`
	Segregation     json.RawMessage `json:"murajaahSegregation"`
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

func GetUserID(telegramID int64) (string, error) {
	rows, err := tablesDB.ListRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithListRowsQueries([]string{
			query.Equal("telegram_id", telegramID),
		}),
	)
	if err != nil {
		return "", err
	}

	var payload Payload

	if err := rows.Decode(&payload); err != nil {
		return "", err
	}
	if len(payload.Rows) == 0 {
		return "", ErrNotFound
	}

	return payload.Rows[0].ID, nil
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

func SaveSunnahSelections(telegramID int64, items []string) ([]string, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	userID, err := GetUserID(telegramID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	date := now.Format("20060102")
	dateWithDashes := now.Format(time.DateOnly)
	rowID := "sun_" + userID + "_" + date

	_, err = tablesDB.UpdateRow(
		config.DatabaseID, config.SunnahLogsTableID,
		rowID,
		tablesDB.WithUpdateRowData(map[string]any{
			"data": string(data),
		}),
	)
	if err != nil {
		_, err = tablesDB.CreateRow(
			config.DatabaseID, config.SunnahLogsTableID,
			rowID,
			map[string]any{
				"$id":         rowID,
				"userId":      userID,
				"data":        string(data),
				"date":        dateWithDashes,
				"telegram_id": telegramID,
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return items, nil
}

func LogMurajaah() error {
	res, err := tablesDB.ListRows(config.DatabaseID, config.UsersTableID)
	if err != nil {
		return err
	}

	yesterday := time.Now().AddDate(0, 0, -1)
	dateCompact := yesterday.Format("20060102")
	dateDisplay := yesterday.Format(time.DateOnly)

	for _, row := range res.Rows {
		var item Row
		item.ID = row.Id
		if err := row.Decode(&item); err != nil {
			continue
		}

		var payload MurajaahPayload
		if err := json.Unmarshal(item.Data, &payload); err != nil {
			continue
		}

		var user struct {
			ID         string `json:"$id"`
			TelegramID int64  `json:"telegram_id"`
		}
		if err := row.Decode(&user); err != nil {
			continue
		}

		rowid := "mur_" + user.ID + "_" + dateCompact
		_, err = tablesDB.UpsertRow(config.DatabaseID, config.MurajaahLogsTableID,
			rowid, tablesDB.WithUpsertRowData(map[string]any{
				"userId":           user.ID,
				"telegram_id":      user.TelegramID,
				"date":             dateDisplay,
				"pageRatings":      string(payload.PageRatings),
				"tasmikRatings":    string(payload.TasmikRatings),
				"repetitionTicks":  string(payload.RepetitionTicks),
				"segregation":      string(payload.Segregation),
			}))
		if err != nil {
			log.Printf("murajaah snapshot: upsert row %s: %v", rowid, err)
			continue
		}
		log.Printf("murajaah snapshot: saved %s", rowid)
	}

	return nil
}

// SetData updates the data column on all rows matching telegram_id.
func SetData(telegramID int64, data map[string]any) error {
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
