package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/pkg/httpx"
	"github.com/memagu/mums/pkg/token"
)

type session struct {
	sync.RWMutex
	userAccountID int64
	expiresAt     time.Time
}

func newSession(userAccountID int64) *session {
	s := &session{userAccountID: userAccountID}
	s.touch()
	return s
}

func (s *session) isExpired() bool {
	s.RLock()
	defer s.RUnlock()
	return s.expiresAt.Before(time.Now())
}

func (s *session) touch() {
	s.Lock()
	defer s.Unlock()
	s.expiresAt = time.Now().Add(config.SessionExpirationTime)
}

type SessionStore struct {
	sync.RWMutex
	sessions map[string]*session // sessionToken -> userAccountID
}

func (ss *SessionStore) cleanupExpiredSessions() {
	ticker := time.NewTicker(config.SessionCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ss.Lock()
		for sessionToken, s := range ss.sessions {
			if s.isExpired() {
				delete(ss.sessions, sessionToken)
			}
		}
		ss.Unlock()
	}
}

func NewSessionStore() *SessionStore {
	ss := &SessionStore{sessions: make(map[string]*session)}

	go ss.cleanupExpiredSessions()

	return ss
}

func (ss *SessionStore) createSession(userAccountID int64) string {
	sessionToken := token.MustGenerateSecure(config.SessionTokenSize)
	s := newSession(userAccountID)

	ss.Lock()
	ss.sessions[sessionToken] = s
	ss.Unlock()

	return sessionToken
}

func (ss *SessionStore) getSession(sessionToken string) (*session, bool) {
	ss.RLock()
	s, ok := ss.sessions[sessionToken]
	ss.RUnlock()

	return s, ok
}

func (ss *SessionStore) deleteSession(sessionToken string) {
	ss.Lock()
	delete(ss.sessions, sessionToken)
	ss.Unlock()
}

func setSessionCookie(c echo.Context, sessionToken string, expiresAt time.Time) {
	sc := new(http.Cookie)
	sc.Name = config.SessionCookieName
	sc.Value = sessionToken
	sc.Path = "/"
	sc.HttpOnly = true
	sc.Secure = true // Set to false for local development, better solution needed
	sc.SameSite = http.SameSiteLaxMode
	sc.Expires = expiresAt
	c.SetCookie(sc)
}

func LoginUser(c echo.Context, ss *SessionStore, userAccountID int64) {
	sessionToken := ss.createSession(userAccountID)

	setSessionCookie(c, sessionToken, time.Now().Add(config.SessionExpirationTime))
}

func SessionMiddleware(ss *SessionStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			setNotLoggedIn := func() error {
				c.Set(config.CTXKeyIsLoggedIn, false)
				return next(c)
			}

			sc, err := c.Cookie(config.SessionCookieName)
			if err != nil {
				return setNotLoggedIn()
			}
			sessionToken := sc.Value

			s, ok := ss.getSession(sessionToken)
			if !ok {
				return setNotLoggedIn()
			}

			if s.isExpired() {
				// Do *NOT* use SessionStore.deleteSession. Deletion is
				// handled by CleanupExpiredSessionsSweeper!
				return setNotLoggedIn()
			}
			s.touch()
			setSessionCookie(c, sessionToken, s.expiresAt)
			
			c.Set(config.CTXKeySessionToken, sessionToken)
			c.Set(config.CTXKeyIsLoggedIn, true)
			c.Set(config.CTXKeyUserAccountID, s.userAccountID)

			return next(c)
		}
	}
}

func getSessionToken(c echo.Context) string {
	sessionToken, ok := c.Get(config.CTXKeySessionToken).(string)
	if !ok {
		panic("config.CTXKeySessionToken is not set in context, was SessionMiddleware not applied?")
	}

	return sessionToken
}

func GetIsLoggedIn(c echo.Context) bool {
	isLoggedIn, ok := c.Get(config.CTXKeyIsLoggedIn).(bool)
	if !ok {
		panic("config.CTXKeyIsLoggedIn is not set in context, was SessionMiddleware not applied?")
	}

	return isLoggedIn
}

func GetUserAccountID(c echo.Context) int64 {
	userAccountID, ok := c.Get(config.CTXKeyUserAccountID).(int64)
	if !ok {
		panic("config.CTXKeyUserAccountID is not set in context, was SessionMiddleware not applied?")
	}

	return userAccountID
}

func RequireSession() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if GetIsLoggedIn(c) {
				return next(c)
			}

			return httpx.Redirect(c, http.StatusSeeOther, "/login")
		}
	}
}

func LogoutUser(c echo.Context, ss *SessionStore) {
	ss.deleteSession(getSessionToken(c))

	setSessionCookie(c, "", time.Unix(0, 0))
}
