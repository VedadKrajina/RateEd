package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type SessionData struct {
	UserID  int64
	IsAdmin bool
}

var (
	sessions   = map[string]SessionData{}
	sessionsMu sync.RWMutex
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func createSession(userID int64, username string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	sessionsMu.Lock()
	sessions[token] = SessionData{
		UserID:  userID,
		IsAdmin: username == adminUsername,
	}
	sessionsMu.Unlock()

	return token
}

func deleteSession(token string) {
	sessionsMu.Lock()
	delete(sessions, token)
	sessionsMu.Unlock()
}

func getSessionFromRequest(r *http.Request) (SessionData, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return SessionData{}, false
	}
	sessionsMu.RLock()
	sd, ok := sessions[cookie.Value]
	sessionsMu.RUnlock()
	return sd, ok
}

func getUserIDFromRequest(r *http.Request) (int64, bool) {
	sd, ok := getSessionFromRequest(r)
	return sd.UserID, ok
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getUserIDFromRequest(r); !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sd, ok := getSessionFromRequest(r)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if !sd.IsAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func isModeratorOrAdmin(r *http.Request, instID int64) bool {
	sd, ok := getSessionFromRequest(r)
	if !ok {
		return false
	}
	if sd.IsAdmin {
		return true
	}
	modID, _ := getInstitutionModerator(instID)
	return modID == sd.UserID
}
