package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
	mu sync.Mutex // serializes writes (sqlite limitation)
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE COLLATE NOCASE,
			name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			user_agent TEXT,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);`,
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			prefix TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			last_used INTEGER,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_user ON api_tokens(user_id);`,
		`CREATE TABLE IF NOT EXISTS forms (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			fields TEXT NOT NULL DEFAULT '[]',
			theme TEXT NOT NULL DEFAULT '{}',
			notifications TEXT NOT NULL DEFAULT '{}',
			accepting INTEGER NOT NULL DEFAULT 1,
			max_responses INTEGER,
			closed_message TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_forms_user ON forms(user_id);`,
		// Best-effort migrations for older databases (ignore errors when columns exist)
		`ALTER TABLE forms ADD COLUMN theme TEXT NOT NULL DEFAULT '{}';`,
		`ALTER TABLE forms ADD COLUMN notifications TEXT NOT NULL DEFAULT '{}';`,
		`ALTER TABLE forms ADD COLUMN max_responses INTEGER;`,
		`ALTER TABLE forms ADD COLUMN closed_message TEXT NOT NULL DEFAULT '';`,
		`CREATE TABLE IF NOT EXISTS responses (
			id TEXT PRIMARY KEY,
			form_id TEXT NOT NULL,
			data TEXT NOT NULL,
			ip TEXT,
			submitted_at INTEGER NOT NULL,
			FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_responses_form ON responses(form_id);`,
		`CREATE TABLE IF NOT EXISTS login_attempts (
			ip TEXT NOT NULL,
			at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_attempts_ip ON login_attempts(ip);`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			// ALTER TABLE statements may fail on new DBs because columns already exist.
			// Tolerate "duplicate column name" and "no such table" errors silently.
			if strings.Contains(err.Error(), "duplicate column") ||
				strings.Contains(err.Error(), "no such table") {
				continue
			}
			return fmt.Errorf("migrate: %w (sql: %s)", err, q)
		}
	}
	return nil
}

// ===== ID helpers =====

func RandomID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func ShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	rand.Read(b)
	out := make([]byte, 10)
	for i, v := range b {
		out[i] = charset[int(v)%len(charset)]
	}
	return string(out)
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// ===== Users =====

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"createdAt"`
}

var ErrNotFound = errors.New("not found")

func (s *Store) UserCount() (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM users`)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) CreateUser(email, name, passwordHash, role string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	res, err := s.db.Exec(
		`INSERT INTO users(email,name,password_hash,role,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
		email, name, passwordHash, role, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Email: email, Name: name, Role: role, CreatedAt: now}, nil
}

func (s *Store) GetUserByEmail(email string) (*User, string, error) {
	row := s.db.QueryRow(`SELECT id,email,name,password_hash,role,created_at FROM users WHERE email=? COLLATE NOCASE`, email)
	var u User
	var hash string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &hash, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return &u, hash, nil
}

func (s *Store) GetUser(id int64) (*User, error) {
	row := s.db.QueryRow(`SELECT id,email,name,role,created_at FROM users WHERE id=?`, id)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id,email,name,role,created_at FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) UpdateUser(id int64, name, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE users SET name=?, role=?, updated_at=? WHERE id=?`,
		name, role, time.Now().Unix(), id)
	return err
}

func (s *Store) UpdateUserPassword(id int64, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE id=?`,
		passwordHash, time.Now().Unix(), id)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

func (s *Store) GetUserPasswordHash(id int64) (string, error) {
	var h string
	err := s.db.QueryRow(`SELECT password_hash FROM users WHERE id=?`, id).Scan(&h)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return h, err
}

// ===== Sessions =====

type Session struct {
	Token     string
	UserID    int64
	UserAgent string
	CreatedAt int64
	ExpiresAt int64
}

const SessionDuration = 30 * 24 * time.Hour

func (s *Store) CreateSession(userID int64, userAgent string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token := RandomID(32)
	now := time.Now().Unix()
	exp := time.Now().Add(SessionDuration).Unix()
	if len(userAgent) > 200 {
		userAgent = userAgent[:200]
	}
	_, err := s.db.Exec(`INSERT INTO sessions(token,user_id,user_agent,created_at,expires_at) VALUES(?,?,?,?,?)`,
		token, userID, userAgent, now, exp)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) GetSession(token string) (*Session, error) {
	row := s.db.QueryRow(`SELECT token,user_id,user_agent,created_at,expires_at FROM sessions WHERE token=?`, token)
	var sess Session
	err := row.Scan(&sess.Token, &sess.UserID, &sess.UserAgent, &sess.CreatedAt, &sess.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if sess.ExpiresAt < time.Now().Unix() {
		s.DeleteSession(token)
		return nil, ErrNotFound
	}
	return &sess, nil
}

func (s *Store) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token=?`, token)
	return err
}

func (s *Store) DeleteUserSessions(userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id=?`, userID)
	return err
}

func (s *Store) PruneSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().Unix())
	s.db.Exec(`DELETE FROM login_attempts WHERE at < ?`, time.Now().Add(-2*time.Hour).Unix())
}

// ===== API Tokens =====

type APIToken struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"-"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	LastUsed  *int64 `json:"lastUsed"`
	CreatedAt int64  `json:"createdAt"`
}

func (s *Store) CreateAPIToken(userID int64, name string) (*APIToken, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw := "gft_" + RandomID(24)
	prefix := raw[:11]
	hash := hashToken(raw)
	now := time.Now().Unix()
	res, err := s.db.Exec(`INSERT INTO api_tokens(user_id,name,prefix,token_hash,created_at) VALUES(?,?,?,?,?)`,
		userID, name, prefix, hash, now)
	if err != nil {
		return nil, "", err
	}
	id, _ := res.LastInsertId()
	return &APIToken{ID: id, UserID: userID, Name: name, Prefix: prefix, CreatedAt: now}, raw, nil
}

func (s *Store) ListAPITokens(userID int64) ([]APIToken, error) {
	rows, err := s.db.Query(`SELECT id,user_id,name,prefix,last_used,created_at FROM api_tokens WHERE user_id=? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIToken
	for rows.Next() {
		var t APIToken
		var lastUsed sql.NullInt64
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Prefix, &lastUsed, &t.CreatedAt); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			t.LastUsed = &lastUsed.Int64
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) DeleteAPIToken(userID, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM api_tokens WHERE id=? AND user_id=?`, id, userID)
	return err
}

func (s *Store) ResolveAPIToken(raw string) (*User, error) {
	hash := hashToken(raw)
	row := s.db.QueryRow(`SELECT u.id,u.email,u.name,u.role,u.created_at,t.id
		FROM api_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=?`, hash)
	var u User
	var tokenID int64
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt, &tokenID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.db.Exec(`UPDATE api_tokens SET last_used=? WHERE id=?`, time.Now().Unix(), tokenID)
	}()
	return &u, nil
}

// ===== Forms =====

type Form struct {
	ID            string         `json:"id"`
	UserID        int64          `json:"userId"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Fields        []any          `json:"fields"`
	Theme         map[string]any `json:"theme"`
	Notifications map[string]any `json:"notifications,omitempty"`
	Accepting     bool           `json:"accepting"`
	MaxResponses  *int           `json:"maxResponses"`
	ClosedMessage string         `json:"closedMessage"`
	CreatedAt     int64          `json:"createdAt"`
	UpdatedAt     int64          `json:"updatedAt"`
	ResponseCount int            `json:"responseCount,omitempty"`
	Owner         *FormOwnerInfo `json:"owner,omitempty"`
}

type FormOwnerInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (s *Store) CreateForm(userID int64, title, description, fieldsJSON, themeJSON, notifJSON string, maxResponses *int, closedMessage string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := ShortID()
	now := time.Now().Unix()
	if themeJSON == "" {
		themeJSON = "{}"
	}
	if notifJSON == "" {
		notifJSON = "{}"
	}
	_, err := s.db.Exec(`INSERT INTO forms(id,user_id,title,description,fields,theme,notifications,accepting,max_responses,closed_message,created_at,updated_at) VALUES(?,?,?,?,?,?,?,1,?,?,?,?)`,
		id, userID, title, description, fieldsJSON, themeJSON, notifJSON, maxResponses, closedMessage, now, now)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) GetForm(id string) (*Form, string, string, string, error) {
	row := s.db.QueryRow(`SELECT id,user_id,title,description,fields,COALESCE(theme,'{}'),COALESCE(notifications,'{}'),accepting,max_responses,COALESCE(closed_message,''),created_at,updated_at FROM forms WHERE id=?`, id)
	var f Form
	var fields, themeStr, notifStr string
	var accepting int
	var maxResp sql.NullInt64
	err := row.Scan(&f.ID, &f.UserID, &f.Title, &f.Description, &fields, &themeStr, &notifStr, &accepting, &maxResp, &f.ClosedMessage, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, "", "", "", ErrNotFound
	}
	if err != nil {
		return nil, "", "", "", err
	}
	f.Accepting = accepting == 1
	if maxResp.Valid {
		v := int(maxResp.Int64)
		f.MaxResponses = &v
	}
	return &f, fields, themeStr, notifStr, nil
}

func (s *Store) ListUserForms(userID int64, all bool) ([]Form, []string, error) {
	var rows *sql.Rows
	var err error
	if all {
		rows, err = s.db.Query(`SELECT f.id,f.user_id,f.title,f.description,f.fields,COALESCE(f.theme,'{}'),f.accepting,f.max_responses,COALESCE(f.closed_message,''),f.created_at,f.updated_at,
			(SELECT COUNT(*) FROM responses r WHERE r.form_id=f.id) AS cnt,
			u.id, u.name
			FROM forms f JOIN users u ON u.id=f.user_id ORDER BY f.updated_at DESC`)
	} else {
		rows, err = s.db.Query(`SELECT f.id,f.user_id,f.title,f.description,f.fields,COALESCE(f.theme,'{}'),f.accepting,f.max_responses,COALESCE(f.closed_message,''),f.created_at,f.updated_at,
			(SELECT COUNT(*) FROM responses r WHERE r.form_id=f.id) AS cnt,
			u.id, u.name
			FROM forms f JOIN users u ON u.id=f.user_id WHERE f.user_id=? ORDER BY f.updated_at DESC`, userID)
	}
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var forms []Form
	var fields []string
	for rows.Next() {
		var f Form
		var fs, themeStr string
		var accepting int
		var maxResp sql.NullInt64
		var ownerID int64
		var ownerName string
		if err := rows.Scan(&f.ID, &f.UserID, &f.Title, &f.Description, &fs, &themeStr, &accepting, &maxResp, &f.ClosedMessage, &f.CreatedAt, &f.UpdatedAt, &f.ResponseCount, &ownerID, &ownerName); err != nil {
			return nil, nil, err
		}
		f.Accepting = accepting == 1
		if maxResp.Valid {
			v := int(maxResp.Int64)
			f.MaxResponses = &v
		}
		f.Owner = &FormOwnerInfo{ID: ownerID, Name: ownerName}
		_ = json.Unmarshal([]byte(themeStr), &f.Theme)
		forms = append(forms, f)
		fields = append(fields, fs)
	}
	return forms, fields, rows.Err()
}

func (s *Store) UpdateForm(id, title, description, fieldsJSON, themeJSON, notifJSON string, accepting bool, maxResponses *int, closedMessage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := 0
	if accepting {
		a = 1
	}
	_, err := s.db.Exec(`UPDATE forms SET title=?,description=?,fields=?,theme=?,notifications=?,accepting=?,max_responses=?,closed_message=?,updated_at=? WHERE id=?`,
		title, description, fieldsJSON, themeJSON, notifJSON, a, maxResponses, closedMessage, time.Now().Unix(), id)
	return err
}

func (s *Store) DeleteForm(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM forms WHERE id=?`, id)
	return err
}

func (s *Store) FormResponseCount(formID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM responses WHERE form_id=?`, formID).Scan(&n)
	return n, err
}

// ===== Responses =====

type Response struct {
	ID          string `json:"id"`
	FormID      string `json:"formId"`
	Data        any    `json:"data"`
	SubmittedAt int64  `json:"submittedAt"`
}

func (s *Store) CreateResponse(formID, dataJSON, ip string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := RandomID(8)
	now := time.Now().Unix()
	_, err := s.db.Exec(`INSERT INTO responses(id,form_id,data,ip,submitted_at) VALUES(?,?,?,?,?)`,
		id, formID, dataJSON, ip, now)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) ListResponses(formID string) ([]string, []string, []int64, error) {
	rows, err := s.db.Query(`SELECT id,data,submitted_at FROM responses WHERE form_id=? ORDER BY submitted_at DESC`, formID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	var ids, datas []string
	var times []int64
	for rows.Next() {
		var id, d string
		var t int64
		if err := rows.Scan(&id, &d, &t); err != nil {
			return nil, nil, nil, err
		}
		ids = append(ids, id)
		datas = append(datas, d)
		times = append(times, t)
	}
	return ids, datas, times, rows.Err()
}

func (s *Store) DeleteResponse(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var formID string
	err := s.db.QueryRow(`SELECT form_id FROM responses WHERE id=?`, id).Scan(&formID)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`DELETE FROM responses WHERE id=?`, id)
	return formID, err
}

// ===== Login attempts (rate limit) =====

func (s *Store) RecordLoginAttempt(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec(`INSERT INTO login_attempts(ip,at) VALUES(?,?)`, ip, time.Now().Unix())
}

func (s *Store) CountLoginAttempts(ip string, since time.Duration) int {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM login_attempts WHERE ip=? AND at >= ?`,
		ip, time.Now().Add(-since).Unix()).Scan(&n)
	return n
}

func (s *Store) ClearLoginAttempts(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec(`DELETE FROM login_attempts WHERE ip=?`, ip)
}
