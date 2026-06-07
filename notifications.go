package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// NotificationConfig is the structured shape of forms.notifications JSON.
// All fields are optional; only enabled channels are dispatched.
type NotificationConfig struct {
	Discord struct {
		Enabled    bool   `json:"enabled"`
		WebhookURL string `json:"webhookUrl"`
	} `json:"discord"`
	Telegram struct {
		Enabled  bool   `json:"enabled"`
		BotToken string `json:"botToken"`
		ChatID   string `json:"chatId"`
	} `json:"telegram"`
	Email struct {
		Enabled  bool   `json:"enabled"`
		SMTPHost string `json:"smtpHost"`
		SMTPPort int    `json:"smtpPort"`
		SMTPUser string `json:"smtpUser"`
		SMTPPass string `json:"smtpPass"`
		From     string `json:"from"`
		To       string `json:"to"`
	} `json:"email"`
}

func parseNotificationConfig(raw string) NotificationConfig {
	var cfg NotificationConfig
	if raw == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(raw), &cfg)
	return cfg
}

// dispatchNotifications fires off all enabled channels in goroutines.
// Each channel has its own timeout; failures are logged, never bubbled up.
func dispatchNotifications(form *Form, cfg NotificationConfig, response map[string]any, formURL string) {
	if cfg.Discord.Enabled && cfg.Discord.WebhookURL != "" {
		go func() {
			if err := sendDiscord(cfg.Discord.WebhookURL, form, response, formURL); err != nil {
				log.Printf("notif discord: %v", err)
			}
		}()
	}
	if cfg.Telegram.Enabled && cfg.Telegram.BotToken != "" && cfg.Telegram.ChatID != "" {
		go func() {
			if err := sendTelegram(cfg.Telegram.BotToken, cfg.Telegram.ChatID, form, response, formURL); err != nil {
				log.Printf("notif telegram: %v", err)
			}
		}()
	}
	if cfg.Email.Enabled && cfg.Email.SMTPHost != "" && cfg.Email.To != "" {
		go func() {
			if err := sendEmail(cfg.Email, form, response, formURL); err != nil {
				log.Printf("notif email: %v", err)
			}
		}()
	}
}

// ===== Helpers =====

// fieldLookup builds id->label map for friendlier output
func fieldLookup(form *Form) map[string]string {
	m := map[string]string{}
	for _, raw := range form.Fields {
		f, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := f["id"].(string)
		label, _ := f["label"].(string)
		if id != "" {
			m[id] = label
		}
	}
	return m
}

func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "—"
	case string:
		if x == "" {
			return "—"
		}
		return x
	case []any:
		var s []string
		for _, it := range x {
			s = append(s, fmt.Sprintf("%v", it))
		}
		return strings.Join(s, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ===== Discord =====

func sendDiscord(webhookURL string, form *Form, resp map[string]any, formURL string) error {
	labels := fieldLookup(form)
	fields := []map[string]any{}
	for _, k := range sortedKeys(resp) {
		name := labels[k]
		if name == "" {
			name = k
		}
		val := formatValue(resp[k])
		if len(val) > 1000 {
			val = val[:1000] + "…"
		}
		fields = append(fields, map[string]any{
			"name":   name,
			"value":  val,
			"inline": len(val) < 40,
		})
	}
	if len(fields) == 0 {
		fields = append(fields, map[string]any{"name": "—", "value": "(boş)"})
	}

	embed := map[string]any{
		"title":       "Yeni yanıt: " + form.Title,
		"description": "Yeni bir form yanıtı alındı.",
		"color":       6577919, // indigo
		"fields":      fields,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"footer":      map[string]any{"text": "goform · " + form.ID},
	}
	if formURL != "" {
		embed["url"] = formURL
	}

	body, _ := json.Marshal(map[string]any{
		"embeds":   []any{embed},
		"username": "goform",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c := &http.Client{Timeout: 10 * time.Second}
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("discord status %d: %s", res.StatusCode, string(b))
	}
	return nil
}

// ===== Telegram =====

func sendTelegram(botToken, chatID string, form *Form, resp map[string]any, formURL string) error {
	labels := fieldLookup(form)
	var sb strings.Builder
	sb.WriteString("📋 *")
	sb.WriteString(tgEscape(form.Title))
	sb.WriteString("*\n")
	sb.WriteString("_Yeni yanıt alındı_\n\n")
	for _, k := range sortedKeys(resp) {
		name := labels[k]
		if name == "" {
			name = k
		}
		val := formatValue(resp[k])
		if len(val) > 300 {
			val = val[:300] + "…"
		}
		sb.WriteString("*")
		sb.WriteString(tgEscape(name))
		sb.WriteString(":* ")
		sb.WriteString(tgEscape(val))
		sb.WriteString("\n")
	}
	if formURL != "" {
		sb.WriteString("\n[Formu aç](")
		sb.WriteString(formURL)
		sb.WriteString(")")
	}

	endpoint := "https://api.telegram.org/bot" + botToken + "/sendMessage"
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", sb.String())
	data.Set("parse_mode", "MarkdownV2")
	data.Set("disable_web_page_preview", "true")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("telegram status %d: %s", res.StatusCode, string(b))
	}
	return nil
}

// Telegram MarkdownV2 escape
func tgEscape(s string) string {
	specials := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, c := range specials {
		s = strings.ReplaceAll(s, c, "\\"+c)
	}
	return s
}

// ===== SMTP Email =====

func sendEmail(cfg struct {
	Enabled  bool   `json:"enabled"`
	SMTPHost string `json:"smtpHost"`
	SMTPPort int    `json:"smtpPort"`
	SMTPUser string `json:"smtpUser"`
	SMTPPass string `json:"smtpPass"`
	From     string `json:"from"`
	To       string `json:"to"`
}, form *Form, resp map[string]any, formURL string) error {
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}
	from := cfg.From
	if from == "" {
		from = cfg.SMTPUser
	}
	labels := fieldLookup(form)

	var text strings.Builder
	text.WriteString("Yeni yanıt: " + form.Title + "\n")
	text.WriteString("-----------------------------------\n\n")
	for _, k := range sortedKeys(resp) {
		name := labels[k]
		if name == "" {
			name = k
		}
		text.WriteString(name + ": " + formatValue(resp[k]) + "\n")
	}
	if formURL != "" {
		text.WriteString("\nFormu aç: " + formURL + "\n")
	}
	text.WriteString("\n— goform")

	subject := "[goform] Yeni yanıt — " + form.Title
	msg := buildEmail(from, cfg.To, subject, text.String())

	addr := cfg.SMTPHost + ":" + strconv.Itoa(cfg.SMTPPort)
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}
	// Recipients can be comma-separated
	recipients := splitAndTrim(cfg.To, ",")
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients")
	}
	return smtp.SendMail(addr, auth, from, recipients, msg)
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildEmail(from, to, subject, body string) []byte {
	// UTF-8 subject (RFC 2047 B-encoded)
	encoded := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      encoded,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
		"Date":         time.Now().Format(time.RFC1123Z),
	}
	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(k + ": " + v + "\r\n")
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return msg.Bytes()
}
