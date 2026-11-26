package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
)

func csvCell(s string) string {
	s = strings.ToValidUTF8(s, "")
	trimmed := strings.TrimLeftFunc(s, unicode.IsSpace)
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + s
	}
	return s
}

func buildCSV(result searchResult) ([]byte, error) {
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	if err := w.Write([]string{"Name", "Father's name", "Phone", "Alternate phone", "Email", "Document number", "Address", "Alternate address"}); err != nil {
		return nil, err
	}
	for i, hit := range result.Hits.Hits {
		if i >= 5 {
			break
		}
		d := hit.Source
		row := []string{d.Name, d.Fname, d.Mobile, d.Alt, d.Email, d.ID, d.Address, d.AltAddress}
		for i := range row {
			row[i] = csvCell(row[i])
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return out.Bytes(), w.Error()
}

func (a *app) handleReportCallback(ctx context.Context, callbackID string, uid int64, data string) error {
	if !strings.HasPrefix(data, "report:") {
		return nil
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(data, "report:"), 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	// A callback is never authorized by the button data alone.
	var encrypted []byte
	err = a.db.QueryRow(ctx, `SELECT r.encrypted_result FROM search_reports r JOIN bot_users u ON u.telegram_id=r.telegram_id JOIN search_deliveries d ON d.update_id=r.update_id WHERE r.update_id=$1 AND r.telegram_id=$2 AND NOT u.blocked AND r.expires_at>now() AND d.status='delivered'`, id, uid).Scan(&encrypted)
	answer := "Preparing your report"
	if err == pgx.ErrNoRows {
		answer = "Report is unavailable or expired. Please search again."
	} else if err != nil {
		return err
	}
	if e := a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": answer}); e != nil {
		return e
	}
	if err == pgx.ErrNoRows {
		return nil
	}
	pages, err := a.decryptDelivery(uid, id, encrypted)
	if err != nil || len(pages) != 1 {
		return fmt.Errorf("invalid encrypted report")
	}
	var result searchResult
	if json.Unmarshal([]byte(pages[0]), &result) != nil {
		return fmt.Errorf("invalid report data")
	}
	content, err := buildCSV(result)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err = w.WriteField("chat_id", strconv.FormatInt(uid, 10)); err != nil {
		return err
	}
	if err = w.WriteField("caption", "Your search report · up to 5 records"); err != nil {
		return err
	}
	part, err := w.CreateFormFile("document", "search-report.csv")
	if err != nil {
		return err
	}
	if _, err = part.Write(content); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+a.token+"/sendDocument", &body)
	if err != nil {
		return fmt.Errorf("invalid telegram request")
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed")
	}
	defer resp.Body.Close()
	var resultBody struct {
		OK bool `json:"ok"`
	}
	if resp.StatusCode/100 != 2 || json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&resultBody) != nil || !resultBody.OK {
		return fmt.Errorf("telegram did not accept report")
	}
	return nil
}

func (a *app) telegramMethod(ctx context.Context, method string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+a.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("invalid telegram request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed")
	}
	defer resp.Body.Close()
	var data struct {
		OK bool `json:"ok"`
	}
	if resp.StatusCode/100 != 2 || json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&data) != nil || !data.OK {
		return fmt.Errorf("telegram rejected request")
	}
	return nil
}
