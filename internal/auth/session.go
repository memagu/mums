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

const (
	ctxKeySessionToken  = "sessionID"
	ctxKeyUserAccountID = "userAccountID"
	ctxKeyIsLoggedIn    = "isLoggedIn"
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
	s.expiresAt = time.Now().Add(config.Auth.SessionTTL)
}

type SessionStore struct {
	sync.RWMutex
	sessions map[string]*session // sessionToken -> userAccountID
}

func (ss *SessionStore) cleanupExpiredSessions() {
	ticker := time.NewTicker(config.Auth.SessionCleanup)
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
	sessionToken := token.MustGenerateSecure(config.Auth.SessionTokenSize)
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

func (ss *SessionStore) DeleteSessionsByUserAccountID(userAccountID int64) {
	ss.Lock()
	defer ss.Unlock()
	for sessionToken, s := range ss.sessions {
		if s.userAccountID == userAccountID {
			delete(ss.sessions, sessionToken)
		}
	}
}

func setSessionCookie(c echo.Context, sessionToken string, ttl time.Duration) {
	httpx.SetCookie(c, config.Auth.SessionCookie, sessionToken, int(ttl.Seconds()), config.Server.CookieSecure)
}

func clearSessionCookie(c echo.Context) {
	httpx.ClearCookie(c, config.Auth.SessionCookie, config.Server.CookieSecure)
}

func LoginUser(c echo.Context, ss *SessionStore, userAccountID int64) {
	sessionToken := ss.createSession(userAccountID)

	setSessionCookie(c, sessionToken, config.Auth.SessionTTL)
}

func SessionMiddleware(ss *SessionStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			setNotLoggedIn := func() error {
				c.Set(ctxKeyIsLoggedIn, false)
				return next(c)
			}

			sc, err := c.Cookie(config.Auth.SessionCookie)
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
			setSessionCookie(c, sessionToken, time.Until(s.expiresAt))

			c.Set(ctxKeySessionToken, sessionToken)
			c.Set(ctxKeyIsLoggedIn, true)
			c.Set(ctxKeyUserAccountID, s.userAccountID)

			return next(c)
		}
	}
}

func getSessionToken(c echo.Context) string {
	return httpx.MustGet[string](c, ctxKeySessionToken, "SessionMiddleware")
}

func GetIsLoggedIn(c echo.Context) bool {
	return httpx.MustGet[bool](c, ctxKeyIsLoggedIn, "SessionMiddleware")
}

func GetUserAccountID(c echo.Context) int64 {
	return httpx.MustGet[int64](c, ctxKeyUserAccountID, "SessionMiddleware")
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

func RedirectLoggedIn() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if GetIsLoggedIn(c) {
				return httpx.Redirect(c, http.StatusSeeOther, "/")
			}

			return next(c)
		}
	}
}

func LogoutUser(c echo.Context, ss *SessionStore) {
	ss.deleteSession(getSessionToken(c))

	clearSessionCookie(c)
}
