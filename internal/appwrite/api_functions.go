package appwrite

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/appwrite/sdk-for-go/query"
	"github.com/solarft/mutabaah-bot/internal/config"
)

var ErrNotFound = errors.New("no row found for this telegram username")

// ErrXPUpdate means the sunnah log was saved but the XP/stats update failed.
var ErrXPUpdate = errors.New("sunnah log saved but XP update failed")

// Keep these in sync with src/store.tsx (toggleSunnahTick) in the website:
// 10 XP per practice, plus a 50 XP bonus for reaching all 10 practices.
const (
	XPPerSunnahTick   = 10
	XPFullSunnahBonus = 50
	XPPerLevel        = 200
	SunnahTickTotal   = 10
)

// sunnahSaveMu serialises read-modify-write cycles per process so two
// concurrent submits cannot both apply the same XP delta.
var sunnahSaveMu sync.Mutex

type Row struct {
	ID   string          `json:"$id"`
	Data json.RawMessage `json:"data"`
}

type Payload struct {
	Rows []Row `json:"rows"`
}

type SunnahLog struct {
	ID    string
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
			ID   string          `json:"$id"`
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

		logs = append(logs, SunnahLog{ID: row.ID, Date: row.Date, Items: items})
	}
	return logs, nil
}

// sunnahRow is the subset of a sunnah_logs document needed to reconcile today's
// log and its XP award.
type sunnahRow struct {
	ID     string          `json:"$id"`
	UserID string          `json:"userId"`
	Date   string          `json:"date"`
	Data   json.RawMessage `json:"data"`
}

// findSunnahLogForDate returns the stored items (and row ID) for one user on one
// date. It queries by userId+date and falls back to scanning the table when the
// attributes are not indexed (mirrors the website's fallback).
func findSunnahLogForDate(userID, date string) (string, []string, error) {
	rows, qerr := tablesDB.ListRows(
		config.DatabaseID, config.SunnahLogsTableID,
		tablesDB.WithListRowsQueries([]string{
			query.Equal("userId", userID),
			query.Equal("date", date),
		}),
	)
	if qerr == nil {
		var payload struct {
			Rows []sunnahRow `json:"rows"`
		}
		if err := rows.Decode(&payload); err != nil {
			return "", nil, err
		}
		if len(payload.Rows) > 0 {
			row := payload.Rows[len(payload.Rows)-1]
			items, err := ParseList(row.Data)
			if err != nil {
				return "", nil, err
			}
			return row.ID, items, nil
		}
		return "", nil, nil
	}

	all, aerr := tablesDB.ListRows(config.DatabaseID, config.SunnahLogsTableID)
	if aerr != nil {
		return "", nil, qerr
	}
	var payload struct {
		Rows []sunnahRow `json:"rows"`
	}
	if err := all.Decode(&payload); err != nil {
		return "", nil, err
	}
	var found *sunnahRow
	for i := range payload.Rows {
		if payload.Rows[i].UserID == userID && payload.Rows[i].Date == date {
			found = &payload.Rows[i]
		}
	}
	if found == nil {
		return "", nil, nil
	}
	items, err := ParseList(found.Data)
	if err != nil {
		return "", nil, err
	}
	return found.ID, items, nil
}

// uniqueCount counts distinct non-empty practice names, matching the website's
// Object.keys(ticks).length semantics.
func uniqueCount(items []string) int {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item != "" {
			seen[item] = struct{}{}
		}
	}
	return len(seen)
}

// sunnahXPDelta mirrors the website's toggleSunnahTick: +10 per practice added,
// -10 per practice removed, +50 when the log reaches all 10 practices and -50
// when it drops below that. Taking a whole-selection delta (instead of
// re-awarding per submit) makes repeated submits idempotent and un-exploitable.
func sunnahXPDelta(oldCount, newCount int) int {
	delta := (newCount - oldCount) * XPPerSunnahTick
	if oldCount != SunnahTickTotal && newCount == SunnahTickTotal {
		delta += XPFullSunnahBonus
	} else if oldCount == SunnahTickTotal && newCount != SunnahTickTotal {
		delta -= XPFullSunnahBonus
	}
	return delta
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// updateStreakStats mirrors updateStreak() in the website's src/utils.ts.
func updateStreakStats(stats map[string]any, now time.Time) {
	today := now.Format(time.DateOnly)
	last, _ := stats["lastActiveDate"].(string)
	if last == "" {
		stats["currentStreak"] = 1
		stats["longestStreak"] = 1
		stats["lastActiveDate"] = today
		return
	}

	parsed, err := time.Parse(time.DateOnly, last)
	if err != nil {
		stats["currentStreak"] = 1
		stats["lastActiveDate"] = today
		return
	}

	lastDay := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, now.Location())
	todayDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Round rather than truncate: DST transitions make a calendar day 23 or 25
	// hours long.
	switch int(math.Round(todayDay.Sub(lastDay).Hours() / 24)) {
	case 0:
		return // already active today
	case 1:
		current := int(toFloat(stats["currentStreak"])) + 1
		longest := int(toFloat(stats["longestStreak"]))
		if current > longest {
			longest = current
		}
		stats["currentStreak"] = current
		stats["longestStreak"] = longest
		stats["lastActiveDate"] = today
	default:
		stats["currentStreak"] = 1
		stats["lastActiveDate"] = today
	}
}

// applySunnahXP adjusts the user's stored stats by the net change in today's
// completed practices, then applies the same level and streak rules as the
// website. It returns the XP delta, the new total, and the new level.
func applySunnahXP(telegramID int64, oldCount, newCount int, now time.Time) (int, float64, int, error) {
	delta := sunnahXPDelta(oldCount, newCount)

	data, err := GetData(telegramID)
	if err != nil {
		return 0, 0, 0, err
	}

	stats, ok := data["stats"].(map[string]any)
	if !ok || stats == nil {
		stats = map[string]any{
			"xp":             float64(0),
			"level":          float64(1),
			"currentStreak":  float64(0),
			"longestStreak":  float64(0),
			"lastActiveDate": "",
		}
		data["stats"] = stats
	}

	xp := math.Max(0, toFloat(stats["xp"])+float64(delta))
	level := int(math.Floor(xp/XPPerLevel)) + 1
	stats["xp"] = xp
	stats["level"] = level
	updateStreakStats(stats, now)

	if err := SetData(telegramID, data); err != nil {
		return 0, 0, 0, err
	}
	return delta, xp, level, nil
}

// SaveSunnahSelections stores today's checklist and reconciles the user's XP by
// the net change against the log already persisted for today. The returned int
// is the XP delta applied (negative when practices were removed).
func SaveSunnahSelections(telegramID int64, items []string) ([]string, int, error) {
	sunnahSaveMu.Lock()
	defer sunnahSaveMu.Unlock()

	data, err := json.Marshal(items)
	if err != nil {
		return nil, 0, err
	}

	userID, err := GetUserID(telegramID)
	if err != nil {
		return nil, 0, err
	}

	now := time.Now()
	date := now.Format("20060102")
	dateWithDashes := now.Format(time.DateOnly)
	rowID := "sun_" + userID + "_" + date

	// Read today's persisted selection before overwriting it. XP is awarded on
	// the difference, so re-submitting the same list never re-awards XP.
	existingRowID, oldItems, err := findSunnahLogForDate(userID, dateWithDashes)
	if err != nil {
		return nil, 0, err
	}

	updateTarget := rowID
	if existingRowID != "" {
		updateTarget = existingRowID
	}

	_, err = tablesDB.UpdateRow(
		config.DatabaseID, config.SunnahLogsTableID,
		updateTarget,
		tablesDB.WithUpdateRowData(map[string]any{
			"data": string(data),
		}),
	)
	if err != nil {
		if existingRowID != "" {
			// The row was listed but could not be updated; creating another row
			// would duplicate today's log and desynchronise the XP baseline.
			return nil, 0, err
		}
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
			return nil, 0, err
		}
	}

	delta, _, _, err := applySunnahXP(telegramID, uniqueCount(oldItems), uniqueCount(items), now)
	if err != nil {
		return items, 0, fmt.Errorf("%w: %v", ErrXPUpdate, err)
	}

	return items, delta, nil
}

func LogMurajaah() error {
	res, err := tablesDB.ListRows(config.DatabaseID, config.UsersTableID)
	if err != nil {
		return err
	}

	var payload struct {
		Rows []struct {
			ID         string          `json:"$id"`
			Data       json.RawMessage `json:"data"`
			TelegramID int64           `json:"telegram_id"`
			UserID     string          `json:"userId"`
		} `json:"rows"`
	}
	if err := res.Decode(&payload); err != nil {
		return err
	}

	yesterday := time.Now().AddDate(0, 0, -1)
	dateCompact := yesterday.Format("20060102")
	dateDisplay := yesterday.Format(time.DateOnly)

	for _, user := range payload.Rows {
		data, err := parseData(user.Data)
		if err != nil {
			log.Printf("murajaah snapshot: parse data for %s: %v", user.ID, err)
			continue
		}

		raw, err := json.Marshal(data)
		if err != nil {
			log.Printf("murajaah snapshot: marshal data for %s: %v", user.ID, err)
			continue
		}

		var mp MurajaahPayload
		if err := json.Unmarshal(raw, &mp); err != nil {
			log.Printf("murajaah snapshot: decode murajaah for %s: %v", user.ID, err)
			continue
		}

		rowid := "mur_" + user.ID + "_" + dateCompact
		_, err = tablesDB.UpsertRow(config.DatabaseID, config.MurajaahLogsTableID,
			rowid, tablesDB.WithUpsertRowData(map[string]any{
				"userId":          user.ID,
				"telegram_id":     user.TelegramID,
				"date":            dateDisplay,
				"pageRatings":     string(mp.PageRatings),
				"tasmikRatings":   string(mp.TasmikRatings),
				"repetitionTicks": string(mp.RepetitionTicks),
				"segregation":     string(mp.Segregation),
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

// SetTelegramID updates the telegram_id column on all rows matching telegram_username,
// then backfills telegram_id onto that user's existing sunnah_logs and murajaah_logs
// rows so pre-link history becomes visible to the bot's telegram_id queries.
func SetTelegramID(username string, telegramID int64) error {
	_, err := tablesDB.UpdateRows(
		config.DatabaseID, config.UsersTableID,
		tablesDB.WithUpdateRowsData(map[string]any{"telegram_id": telegramID}),
		tablesDB.WithUpdateRowsQueries([]string{
			query.Equal("telegram_username", username),
		}),
	)
	if err != nil {
		return err
	}

	// Backfill telegram_id onto log rows belonging to this user.
	userID, err := GetUserID(telegramID)
	if err != nil {
		log.Printf("appwrite backfill: get user id for telegram_id %d: %v", telegramID, err)
		return nil
	}

	for _, table := range []string{config.SunnahLogsTableID, config.MurajaahLogsTableID} {
		if _, err := tablesDB.UpdateRows(
			config.DatabaseID, table,
			tablesDB.WithUpdateRowsData(map[string]any{"telegram_id": telegramID}),
			tablesDB.WithUpdateRowsQueries([]string{
				query.Equal("userId", userID),
			}),
		); err != nil {
			log.Printf("appwrite backfill: update %s rows for user %s: %v", table, userID, err)
		}
	}

	return nil
}
