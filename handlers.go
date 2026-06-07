package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	Store     *Store
	Pages     fs.FS
	PublicURL string
}

func (s *Server) Routes(mux *http.ServeMux, staticFS fs.FS) {
	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// HTML pages
	mux.HandleFunc("GET /{$}", s.pageHome)
	mux.HandleFunc("GET /login", s.pageLogin)
	mux.HandleFunc("GET /setup", s.pageSetup)
	mux.HandleFunc("GET /dashboard", s.requireHTML(s.servePage("dashboard.html")))
	mux.HandleFunc("GET /build/{id}", s.requireHTML(s.servePage("builder.html")))
	mux.HandleFunc("GET /responses/{id}", s.requireHTML(s.servePage("responses.html")))
	mux.HandleFunc("GET /settings", s.requireHTML(s.servePage("settings.html")))

	// Public form filling (no auth)
	mux.HandleFunc("GET /f/{id}", s.servePage("view.html"))

	// === Auth API ===
	mux.HandleFunc("POST /api/auth/setup", s.apiSetup)
	mux.HandleFunc("POST /api/auth/login", s.apiLogin)
	mux.HandleFunc("POST /api/auth/logout", s.apiLogout)
	mux.HandleFunc("GET /api/auth/me", s.requireAuth(s.apiMe))
	mux.HandleFunc("POST /api/auth/change-password", s.requireAuth(s.apiChangePassword))

	// === User management (admin) ===
	mux.HandleFunc("GET /api/users", s.requireAdmin(s.apiListUsers))
	mux.HandleFunc("POST /api/users", s.requireAdmin(s.apiCreateUser))
	mux.HandleFunc("PUT /api/users/{id}", s.requireAdmin(s.apiUpdateUser))
	mux.HandleFunc("POST /api/users/{id}/reset-password", s.requireAdmin(s.apiResetPassword))
	mux.HandleFunc("DELETE /api/users/{id}", s.requireAdmin(s.apiDeleteUser))

	// === API tokens ===
	mux.HandleFunc("GET /api/tokens", s.requireAuth(s.apiListTokens))
	mux.HandleFunc("POST /api/tokens", s.requireAuth(s.apiCreateToken))
	mux.HandleFunc("DELETE /api/tokens/{id}", s.requireAuth(s.apiDeleteToken))

	// === Forms ===
	mux.HandleFunc("GET /api/forms", s.requireAuth(s.apiListForms))
	mux.HandleFunc("POST /api/forms", s.requireAuth(s.apiCreateForm))
	mux.HandleFunc("GET /api/forms/{id}", s.requireAuth(s.apiGetForm))
	mux.HandleFunc("PUT /api/forms/{id}", s.requireAuth(s.apiUpdateForm))
	mux.HandleFunc("DELETE /api/forms/{id}", s.requireAuth(s.apiDeleteForm))

	// === Notifications (sensitive: own/admin only) ===
	mux.HandleFunc("GET /api/forms/{id}/notifications", s.requireAuth(s.apiGetNotifications))
	mux.HandleFunc("PUT /api/forms/{id}/notifications", s.requireAuth(s.apiUpdateNotifications))
	mux.HandleFunc("POST /api/forms/{id}/notifications/test", s.requireAuth(s.apiTestNotifications))

	// === Public form (no auth) ===
	mux.HandleFunc("GET /api/public/forms/{id}", s.apiPublicForm)
	mux.HandleFunc("POST /api/public/forms/{id}/responses", s.apiSubmitResponse)

	// === Responses ===
	mux.HandleFunc("GET /api/forms/{id}/responses", s.requireAuth(s.apiListResponses))
	mux.HandleFunc("DELETE /api/responses/{id}", s.requireAuth(s.apiDeleteResponse))

	// === Status ===
	mux.HandleFunc("GET /api/status", s.apiStatus)
	mux.HandleFunc("GET /api/help", s.apiHelp)
}

// ===== JSON helpers =====

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(body)
}

func readJSON(r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 4<<20) // 4 MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func badReq(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

// ===== Page handlers =====

func (s *Server) pageHome(w http.ResponseWriter, r *http.Request) {
	count, _ := s.Store.UserCount()
	if count == 0 {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	if u := s.authenticate(r); u != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) pageLogin(w http.ResponseWriter, r *http.Request) {
	count, _ := s.Store.UserCount()
	if count == 0 {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	if u := s.authenticate(r); u != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	s.servePage("login.html")(w, r)
}

func (s *Server) pageSetup(w http.ResponseWriter, r *http.Request) {
	count, _ := s.Store.UserCount()
	if count > 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	s.servePage("setup.html")(w, r)
}

func (s *Server) servePage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := s.Pages.Open(name)
		if err != nil {
			http.Error(w, "page not found", 404)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		io.Copy(w, f)
	}
}

// ===== Auth handlers =====

type setupReq struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (s *Server) apiSetup(w http.ResponseWriter, r *http.Request) {
	count, _ := s.Store.UserCount()
	if count > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "setup already completed"})
		return
	}
	var req setupReq
	if err := readJSON(r, &req); err != nil {
		badReq(w, "invalid payload")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if !validEmail(req.Email) {
		badReq(w, "invalid email")
		return
	}
	if req.Name == "" {
		req.Name = req.Email
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		badReq(w, err.Error())
		return
	}
	user, err := s.Store.CreateUser(req.Email, req.Name, hash, RoleAdmin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create user"})
		return
	}
	token, _ := s.Store.CreateSession(user.ID, r.UserAgent())
	setSessionCookie(w, token, isHTTPS(r))
	writeJSON(w, http.StatusOK, user)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) apiLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if s.Store.CountLoginAttempts(ip, 15*time.Minute) >= 10 {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts, please wait"})
		return
	}
	var req loginReq
	if err := readJSON(r, &req); err != nil {
		badReq(w, "invalid payload")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	user, hash, err := s.Store.GetUserByEmail(req.Email)
	if err != nil || !checkPassword(hash, req.Password) {
		s.Store.RecordLoginAttempt(ip)
		// Constant-ish response time
		time.Sleep(200 * time.Millisecond)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "e-posta veya şifre hatalı"})
		return
	}
	s.Store.ClearLoginAttempts(ip)
	token, err := s.Store.CreateSession(user.ID, r.UserAgent())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create session"})
		return
	}
	setSessionCookie(w, token, isHTTPS(r))
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) apiLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookie); err == nil {
		s.Store.DeleteSession(c.Value)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) apiMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userFromCtx(r.Context()))
}

type changePwReq struct {
	Current string `json:"currentPassword"`
	New     string `json:"newPassword"`
}

func (s *Server) apiChangePassword(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	var req changePwReq
	if err := readJSON(r, &req); err != nil {
		badReq(w, "invalid payload")
		return
	}
	hash, err := s.Store.GetUserPasswordHash(u.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user not found"})
		return
	}
	if !checkPassword(hash, req.Current) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "mevcut şifre yanlış"})
		return
	}
	newHash, err := hashPassword(req.New)
	if err != nil {
		badReq(w, err.Error())
		return
	}
	if err := s.Store.UpdateUserPassword(u.ID, newHash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update password"})
		return
	}
	// Invalidate other sessions
	s.Store.DeleteUserSessions(u.ID)
	token, _ := s.Store.CreateSession(u.ID, r.UserAgent())
	setSessionCookie(w, token, isHTTPS(r))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== User management =====

type createUserReq struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (s *Server) apiListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.ListUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) apiCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if err := readJSON(r, &req); err != nil {
		badReq(w, "invalid payload")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if !validEmail(req.Email) {
		badReq(w, "invalid email")
		return
	}
	if req.Name == "" {
		req.Name = req.Email
	}
	if req.Role != RoleAdmin && req.Role != RoleUser {
		req.Role = RoleUser
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		badReq(w, err.Error())
		return
	}
	if _, _, err := s.Store.GetUserByEmail(req.Email); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "bu e-posta zaten kayıtlı"})
		return
	}
	user, err := s.Store.CreateUser(req.Email, req.Name, hash, req.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type updateUserReq struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func (s *Server) apiUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		badReq(w, "invalid id")
		return
	}
	var req updateUserReq
	if err := readJSON(r, &req); err != nil {
		badReq(w, "invalid payload")
		return
	}
	target, err := s.Store.GetUser(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if req.Role != RoleAdmin && req.Role != RoleUser {
		req.Role = target.Role
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = target.Name
	}
	// Don't allow demoting the last admin
	if target.Role == RoleAdmin && req.Role != RoleAdmin {
		users, _ := s.Store.ListUsers()
		adminCount := 0
		for _, u := range users {
			if u.Role == RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "son admin yetkisi kaldırılamaz"})
			return
		}
	}
	if err := s.Store.UpdateUser(id, name, req.Role); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	u, _ := s.Store.GetUser(id)
	writeJSON(w, http.StatusOK, u)
}

type resetPwReq struct {
	Password string `json:"password"`
}

func (s *Server) apiResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		badReq(w, "invalid id")
		return
	}
	var req resetPwReq
	if err := readJSON(r, &req); err != nil {
		badReq(w, "invalid payload")
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		badReq(w, err.Error())
		return
	}
	if err := s.Store.UpdateUserPassword(id, hash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.Store.DeleteUserSessions(id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) apiDeleteUser(w http.ResponseWriter, r *http.Request) {
	current := userFromCtx(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		badReq(w, "invalid id")
		return
	}
	if id == current.ID {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "kendinizi silemezsiniz"})
		return
	}
	target, err := s.Store.GetUser(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if target.Role == RoleAdmin {
		users, _ := s.Store.ListUsers()
		adminCount := 0
		for _, u := range users {
			if u.Role == RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "son admin silinemez"})
			return
		}
	}
	if err := s.Store.DeleteUser(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== API Tokens =====

type createTokenReq struct {
	Name string `json:"name"`
}

func (s *Server) apiListTokens(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	tokens, err := s.Store.ListAPITokens(u.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tokens == nil {
		tokens = []APIToken{}
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) apiCreateToken(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	var req createTokenReq
	if err := readJSON(r, &req); err != nil {
		badReq(w, "invalid payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = "API Token"
	}
	if len(req.Name) > 80 {
		req.Name = req.Name[:80]
	}
	tok, raw, err := s.Store.CreateAPIToken(u.ID, req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": raw, // only returned once
		"info":  tok,
	})
}

func (s *Server) apiDeleteToken(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		badReq(w, "invalid id")
		return
	}
	if err := s.Store.DeleteAPIToken(u.ID, id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== Forms =====

type formPayload struct {
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Fields        any            `json:"fields"`
	Theme         map[string]any `json:"theme"`
	Accepting     *bool          `json:"accepting"`
	MaxResponses  *int           `json:"maxResponses"`
	ClosedMessage *string        `json:"closedMessage"`
}

func (s *Server) apiListForms(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	all := u.Role == RoleAdmin && r.URL.Query().Get("scope") == "all"
	forms, fieldStrs, err := s.Store.ListUserForms(u.ID, all)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if forms == nil {
		writeJSON(w, http.StatusOK, []Form{})
		return
	}
	for i := range forms {
		json.Unmarshal([]byte(fieldStrs[i]), &forms[i].Fields)
	}
	writeJSON(w, http.StatusOK, forms)
}

func (s *Server) apiCreateForm(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	var p formPayload
	if err := readJSON(r, &p); err != nil {
		p = formPayload{}
	}
	if p.Title == "" {
		p.Title = "İsimsiz form"
	}
	fields := []any{}
	if p.Fields != nil {
		if arr, ok := p.Fields.([]any); ok {
			fields = arr
		}
	}
	fieldsJSON, _ := json.Marshal(fields)
	themeJSON, _ := json.Marshal(p.Theme)
	closedMsg := ""
	if p.ClosedMessage != nil {
		closedMsg = *p.ClosedMessage
	}
	id, err := s.Store.CreateForm(u.ID, p.Title, p.Description, string(fieldsJSON), string(themeJSON), "{}", p.MaxResponses, closedMsg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	form, _, themeStr, _, _ := s.Store.GetForm(id)
	json.Unmarshal([]byte(themeStr), &form.Theme)
	form.Fields = fields
	writeJSON(w, http.StatusOK, form)
}

func (s *Server) apiGetForm(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	form, fieldsJSON, themeJSON, _, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	json.Unmarshal([]byte(fieldsJSON), &form.Fields)
	json.Unmarshal([]byte(themeJSON), &form.Theme)
	form.ResponseCount, _ = s.Store.FormResponseCount(id)
	writeJSON(w, http.StatusOK, form)
}

func (s *Server) apiUpdateForm(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	form, existingFields, existingTheme, existingNotif, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	var p formPayload
	if err := readJSON(r, &p); err != nil {
		badReq(w, "invalid payload")
		return
	}
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = form.Title
	}
	desc := p.Description
	fieldsJSON := existingFields
	if p.Fields != nil {
		b, _ := json.Marshal(p.Fields)
		fieldsJSON = string(b)
	}
	themeJSON := existingTheme
	if p.Theme != nil {
		b, _ := json.Marshal(p.Theme)
		themeJSON = string(b)
	}
	accepting := form.Accepting
	if p.Accepting != nil {
		accepting = *p.Accepting
	}
	maxResp := form.MaxResponses
	if p.MaxResponses != nil {
		// Treat 0 or negative as unlimited
		if *p.MaxResponses <= 0 {
			maxResp = nil
		} else {
			maxResp = p.MaxResponses
		}
	}
	closedMsg := form.ClosedMessage
	if p.ClosedMessage != nil {
		closedMsg = *p.ClosedMessage
	}
	if err := s.Store.UpdateForm(id, title, desc, fieldsJSON, themeJSON, existingNotif, accepting, maxResp, closedMsg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	updated, fs, ts, _, _ := s.Store.GetForm(id)
	json.Unmarshal([]byte(fs), &updated.Fields)
	json.Unmarshal([]byte(ts), &updated.Theme)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) apiDeleteForm(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	form, _, _, _, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	if err := s.Store.DeleteForm(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== Notifications =====

func (s *Server) apiGetNotifications(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	form, _, _, notifJSON, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(notifJSON), &out); err != nil {
		out = map[string]any{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) apiUpdateNotifications(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	form, fieldsJSON, themeJSON, _, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	var payload map[string]any
	if err := readJSON(r, &payload); err != nil {
		badReq(w, "invalid payload")
		return
	}
	notifJSON, _ := json.Marshal(payload)
	if err := s.Store.UpdateForm(id, form.Title, form.Description, fieldsJSON, themeJSON, string(notifJSON), form.Accepting, form.MaxResponses, form.ClosedMessage); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) apiTestNotifications(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	form, _, _, notifJSON, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	cfg := parseNotificationConfig(notifJSON)
	// Make sure form.Fields is populated for label lookup
	json.Unmarshal([]byte(notifJSON), &cfg)
	sample := map[string]any{
		"_test": "Bu bir test bildirimidir. Form: " + form.Title,
		"_when": time.Now().Format(time.RFC3339),
	}
	dispatchNotifications(form, cfg, sample, s.formURL(r, id))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== Public form =====

const (
	ErrCodeNotFound      = "not_found"
	ErrCodeFormClosed    = "form_closed"
	ErrCodeMaxReached    = "max_reached"
	ErrCodeUnauthorized  = "unauthorized"
)

func (s *Server) apiPublicForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	form, fieldsJSON, themeJSON, _, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Form bulunamadı",
			"code":  ErrCodeNotFound,
		})
		return
	}
	// Check accepting status
	if !form.Accepting {
		msg := form.ClosedMessage
		if msg == "" {
			msg = "Bu form sahibi tarafından durdurulmuştur."
		}
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":     msg,
			"code":      ErrCodeFormClosed,
			"formTitle": form.Title,
			"theme":     parseJSONMap(themeJSON),
		})
		return
	}
	// Check max responses
	if form.MaxResponses != nil && *form.MaxResponses > 0 {
		count, _ := s.Store.FormResponseCount(id)
		if count >= *form.MaxResponses {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"error":         "Bu form maksimum yanıt sayısına ulaşmıştır.",
				"code":          ErrCodeMaxReached,
				"formTitle":     form.Title,
				"theme":         parseJSONMap(themeJSON),
				"maxResponses":  *form.MaxResponses,
				"responseCount": count,
			})
			return
		}
	}
	json.Unmarshal([]byte(fieldsJSON), &form.Fields)
	json.Unmarshal([]byte(themeJSON), &form.Theme)
	// Don't leak ownership info
	form.UserID = 0
	form.Owner = nil
	form.Notifications = nil
	form.ClosedMessage = "" // not needed in public form
	writeJSON(w, http.StatusOK, form)
}

func parseJSONMap(s string) map[string]any {
	var m map[string]any
	json.Unmarshal([]byte(s), &m)
	return m
}

func (s *Server) apiSubmitResponse(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	form, fieldsJSON, _, notifJSON, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Form bulunamadı", "code": ErrCodeNotFound})
		return
	}
	if !form.Accepting {
		msg := form.ClosedMessage
		if msg == "" {
			msg = "Bu form sahibi tarafından durdurulmuştur."
		}
		writeJSON(w, http.StatusForbidden, map[string]string{"error": msg, "code": ErrCodeFormClosed})
		return
	}
	if form.MaxResponses != nil && *form.MaxResponses > 0 {
		count, _ := s.Store.FormResponseCount(id)
		if count >= *form.MaxResponses {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "Bu form maksimum yanıt sayısına ulaşmıştır.",
				"code":  ErrCodeMaxReached,
			})
			return
		}
	}
	var data map[string]any
	if err := readJSON(r, &data); err != nil {
		badReq(w, "invalid payload")
		return
	}
	b, _ := json.Marshal(data)
	rid, err := s.Store.CreateResponse(id, string(b), clientIP(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Dispatch notifications asynchronously
	cfg := parseNotificationConfig(notifJSON)
	json.Unmarshal([]byte(fieldsJSON), &form.Fields)
	dispatchNotifications(form, cfg, data, s.formURL(r, id))

	writeJSON(w, http.StatusOK, map[string]any{"id": rid, "submittedAt": time.Now().Unix()})
}

// ===== Responses =====

func (s *Server) apiListResponses(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	form, _, _, _, err := s.Store.GetForm(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	ids, datas, times, err := s.Store.ListResponses(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp := make([]map[string]any, 0, len(ids))
	for i, did := range ids {
		var dat any
		json.Unmarshal([]byte(datas[i]), &dat)
		resp = append(resp, map[string]any{
			"id":          did,
			"data":        dat,
			"submittedAt": times[i],
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) apiDeleteResponse(w http.ResponseWriter, r *http.Request) {
	u := userFromCtx(r.Context())
	id := r.PathValue("id")
	var formID string
	row := s.Store.db.QueryRow(`SELECT form_id FROM responses WHERE id=?`, id)
	if err := row.Scan(&formID); err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	form, _, _, _, err := s.Store.GetForm(formID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "form not found"})
		return
	}
	if !canAccessForm(u, form) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "no access"})
		return
	}
	if _, err := s.Store.DeleteResponse(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== Status =====

func (s *Server) apiStatus(w http.ResponseWriter, r *http.Request) {
	count, _ := s.Store.UserCount()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":    Version,
		"setupDone":  count > 0,
		"userCount":  count,
		"authPolicy": "session-cookie or Bearer api_token",
		"publicUrl":  s.publicBaseURL(r),
		"helpUrl":    s.publicBaseURL(r) + "/api/help",
	})
}

// apiHelp returns machine-readable API metadata, intended for AI agents and
// other automation tools. No auth required.
func (s *Server) apiHelp(w http.ResponseWriter, r *http.Request) {
	base := s.publicBaseURL(r)
	help := map[string]any{
		"name":        "goform",
		"version":     Version,
		"baseUrl":     base,
		"description": "Modern form builder with REST API. Supports session-cookie or Bearer token auth.",
		"docs":        "https://github.com/benyigiteren/goform#readme",
		"auth": map[string]any{
			"sessionCookie": map[string]string{
				"cookie":      SessionCookie,
				"description": "Set automatically by POST /api/auth/login. HTTP-only.",
			},
			"bearerToken": map[string]any{
				"header":      "Authorization: Bearer <token>",
				"prefix":      "gft_",
				"createdVia":  "POST /api/tokens",
				"description": "Create tokens at /settings (gear icon → API Erişimi).",
			},
		},
		"fieldTypes": []map[string]any{
			{"type": "short_text", "label": "Short text", "options": []string{"placeholder"}},
			{"type": "long_text", "label": "Long text", "options": []string{"placeholder"}},
			{"type": "email", "label": "Email", "options": []string{"placeholder"}},
			{"type": "url", "label": "URL", "options": []string{"placeholder"}},
			{"type": "phone", "label": "Phone", "options": []string{"placeholder"}},
			{"type": "number", "label": "Number", "options": []string{"placeholder", "min", "max"}},
			{"type": "date", "label": "Date", "options": []string{}},
			{"type": "time", "label": "Time", "options": []string{}},
			{"type": "radio", "label": "Single choice", "options": []string{"options[]"}},
			{"type": "checkbox", "label": "Multiple choice", "options": []string{"options[]"}},
			{"type": "dropdown", "label": "Dropdown", "options": []string{"options[]"}},
			{"type": "rating", "label": "Star rating", "options": []string{"maxStars (3-10)"}},
			{"type": "scale", "label": "Linear scale", "options": []string{"min", "max", "minLabel", "maxLabel"}},
			{"type": "section", "label": "Section header", "options": []string{"description"}},
		},
		"endpoints": []map[string]string{
			// Auth
			{"method": "POST", "path": "/api/auth/setup", "auth": "none", "body": "{email,name,password}", "description": "Create the first super admin"},
			{"method": "POST", "path": "/api/auth/login", "auth": "none", "body": "{email,password}", "description": "Login; sets session cookie"},
			{"method": "POST", "path": "/api/auth/logout", "auth": "none", "description": "Logout, deletes session"},
			{"method": "GET", "path": "/api/auth/me", "auth": "required", "description": "Current user info"},
			{"method": "POST", "path": "/api/auth/change-password", "auth": "required", "body": "{currentPassword,newPassword}", "description": "Change own password"},
			// Forms
			{"method": "GET", "path": "/api/forms", "auth": "required", "query": "?scope=all (admin only)", "description": "List forms"},
			{"method": "POST", "path": "/api/forms", "auth": "required", "body": "{title,description,fields,theme,maxResponses,closedMessage}", "description": "Create form"},
			{"method": "GET", "path": "/api/forms/:id", "auth": "owner/admin", "description": "Get form"},
			{"method": "PUT", "path": "/api/forms/:id", "auth": "owner/admin", "body": "{title,description,fields,theme,accepting,maxResponses,closedMessage}", "description": "Update form"},
			{"method": "DELETE", "path": "/api/forms/:id", "auth": "owner/admin", "description": "Delete form"},
			// Notifications
			{"method": "GET", "path": "/api/forms/:id/notifications", "auth": "owner/admin", "description": "Get notification config"},
			{"method": "PUT", "path": "/api/forms/:id/notifications", "auth": "owner/admin", "body": "{discord:{enabled,webhookUrl}, telegram:{enabled,botToken,chatId}, email:{enabled,smtpHost,smtpPort,smtpUser,smtpPass,from,to}}", "description": "Update notification channels"},
			{"method": "POST", "path": "/api/forms/:id/notifications/test", "auth": "owner/admin", "description": "Send a test notification on each enabled channel"},
			// Public form
			{"method": "GET", "path": "/api/public/forms/:id", "auth": "none", "description": "Read form fields for filling. Returns 403 with code 'form_closed' or 'max_reached' when not accepting."},
			{"method": "POST", "path": "/api/public/forms/:id/responses", "auth": "none", "body": "{<fieldId>:<value>, ...}", "description": "Submit a form response. Triggers notifications."},
			// Responses
			{"method": "GET", "path": "/api/forms/:id/responses", "auth": "owner/admin", "description": "List responses"},
			{"method": "DELETE", "path": "/api/responses/:id", "auth": "owner/admin", "description": "Delete a response"},
			// Tokens
			{"method": "GET", "path": "/api/tokens", "auth": "required", "description": "List my API tokens"},
			{"method": "POST", "path": "/api/tokens", "auth": "required", "body": "{name}", "description": "Create new API token (raw token returned once)"},
			{"method": "DELETE", "path": "/api/tokens/:id", "auth": "required", "description": "Revoke API token"},
			// Users (admin)
			{"method": "GET", "path": "/api/users", "auth": "admin", "description": "List users"},
			{"method": "POST", "path": "/api/users", "auth": "admin", "body": "{email,name,password,role}", "description": "Create user (role: 'user' or 'admin')"},
			{"method": "PUT", "path": "/api/users/:id", "auth": "admin", "body": "{name,role}", "description": "Update user"},
			{"method": "POST", "path": "/api/users/:id/reset-password", "auth": "admin", "body": "{password}", "description": "Reset user's password"},
			{"method": "DELETE", "path": "/api/users/:id", "auth": "admin", "description": "Delete user"},
		},
		"errorCodes": []map[string]string{
			{"code": ErrCodeNotFound, "description": "Form / resource does not exist"},
			{"code": ErrCodeFormClosed, "description": "Form is not accepting responses"},
			{"code": ErrCodeMaxReached, "description": "Max responses limit reached"},
			{"code": ErrCodeUnauthorized, "description": "Auth required or invalid"},
		},
		"examples": map[string]string{
			"createForm": `POST ` + base + `/api/forms
Authorization: Bearer gft_xxx
Content-Type: application/json

{
  "title": "Müşteri Memnuniyet",
  "description": "Geri bildirim",
  "theme": {"color": "rose", "icon": "⭐", "mode": "light"},
  "maxResponses": 100,
  "fields": [
    {"id": "f1", "type": "short_text", "label": "Adınız", "required": true},
    {"id": "f2", "type": "rating", "label": "Puan", "maxStars": 5, "required": true}
  ]
}`,
			"listResponses": `GET ` + base + `/api/forms/<id>/responses
Authorization: Bearer gft_xxx`,
			"submitResponse": `POST ` + base + `/api/public/forms/<id>/responses
Content-Type: application/json

{"f1": "Ali Veli", "f2": 5}`,
		},
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, help)
}

// ===== Helpers =====

// publicBaseURL returns the public base URL.
// Priority: server.PublicURL > X-Forwarded headers > Host header.
func (s *Server) publicBaseURL(r *http.Request) string {
	if s.PublicURL != "" {
		return s.PublicURL
	}
	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func (s *Server) formURL(r *http.Request, formID string) string {
	base := s.publicBaseURL(r)
	if base == "" {
		return ""
	}
	return base + "/f/" + formID
}

func canAccessForm(u *User, f *Form) bool {
	if u == nil || f == nil {
		return false
	}
	if u.Role == RoleAdmin {
		return true
	}
	return f.UserID == u.ID
}

func validEmail(s string) bool {
	if len(s) < 3 || len(s) > 254 {
		return false
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	if strings.IndexByte(s[at+1:], '.') < 0 {
		return false
	}
	return true
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
