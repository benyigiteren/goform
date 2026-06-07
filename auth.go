package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookie = "goform_session"
	RoleAdmin     = "admin"
	RoleUser      = "user"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
)

func userFromCtx(ctx context.Context) *User {
	if v, ok := ctx.Value(ctxKeyUser).(*User); ok {
		return v
	}
	return nil
}

func ctxWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

// ===== Password hashing =====

func hashPassword(pw string) (string, error) {
	if len(pw) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	if len(pw) > 128 {
		return "", errors.New("password is too long")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

func checkPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// ===== Cookie helpers =====

func setSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(SessionDuration.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// ===== Auth middleware =====

// authenticate resolves either a session cookie or an Authorization: Bearer <api_token>.
// Sets the user in the request context if authenticated. Does not block unauthenticated requests.
func (s *Server) authenticate(r *http.Request) *User {
	// Bearer token
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		raw := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		if raw != "" {
			if u, err := s.Store.ResolveAPIToken(raw); err == nil {
				return u
			}
		}
	}
	// Session cookie
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		if sess, err := s.Store.GetSession(c.Value); err == nil {
			if u, err := s.Store.GetUser(sess.UserID); err == nil {
				return u
			}
		}
	}
	return nil
}

// requireAuth wraps a handler and requires authentication.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := s.authenticate(r)
		if u == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		ctx := ctxWithUser(r.Context(), u)
		next(w, r.WithContext(ctx))
	}
}

// requireAdmin wraps a handler and requires admin role.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		u := userFromCtx(r.Context())
		if u == nil || u.Role != RoleAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin privileges required"})
			return
		}
		next(w, r)
	})
}

// requireHTML redirects browser requests to login when unauthenticated.
func (s *Server) requireHTML(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if first user setup is needed
		count, _ := s.Store.UserCount()
		if count == 0 {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}
		u := s.authenticate(r)
		if u == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		ctx := ctxWithUser(r.Context(), u)
		next(w, r.WithContext(ctx))
	}
}

// clientIP extracts the IP for rate limiting.
func clientIP(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-For"); h != "" {
		if idx := strings.Index(h, ","); idx > 0 {
			return strings.TrimSpace(h[:idx])
		}
		return strings.TrimSpace(h)
	}
	if h := r.Header.Get("X-Real-IP"); h != "" {
		return h
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}
