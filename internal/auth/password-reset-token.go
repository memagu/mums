package auth

import (
	"sync"
	"time"

	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/pkg/token"
)

type passwordResetToken struct {
	sync.RWMutex
	userAccountID int64
	expiresAt     time.Time
}

func newPasswordResetToken(userAccountID int64) *passwordResetToken {
	return &passwordResetToken{
		userAccountID: userAccountID,
		expiresAt:     time.Now().Add(config.Auth.PasswordResetTTL),
	}
}

func (t *passwordResetToken) isExpired() bool {
	t.RLock()
	defer t.RUnlock()
	return t.expiresAt.Before(time.Now())
}

type PasswordResetTokenStore struct {
	sync.RWMutex
	tokens map[string]*passwordResetToken // token -> reset token
}

func NewPasswordResetTokenStore() *PasswordResetTokenStore {
	rts := &PasswordResetTokenStore{tokens: make(map[string]*passwordResetToken)}

	go rts.cleanupExpiredTokens()

	return rts
}

func (rts *PasswordResetTokenStore) cleanupExpiredTokens() {
	ticker := time.NewTicker(config.Auth.SessionCleanup)
	defer ticker.Stop()

	for range ticker.C {
		rts.Lock()
		for tokenValue, t := range rts.tokens {
			if t.isExpired() {
				delete(rts.tokens, tokenValue)
			}
		}
		rts.Unlock()
	}
}

func (rts *PasswordResetTokenStore) Create(userAccountID int64) string {
	tokenValue := token.MustGenerateSecure(config.Auth.PasswordResetTokenSize)
	t := newPasswordResetToken(userAccountID)

	rts.Lock()
	rts.tokens[tokenValue] = t
	rts.Unlock()

	return tokenValue
}

func (rts *PasswordResetTokenStore) Peek(tokenValue string) (int64, bool) {
	rts.RLock()
	t, ok := rts.tokens[tokenValue]
	rts.RUnlock()

	if !ok || t.isExpired() {
		return 0, false
	}

	return t.userAccountID, true
}

func (rts *PasswordResetTokenStore) Consume(tokenValue string) (int64, bool) {
	rts.Lock()
	defer rts.Unlock()

	t, ok := rts.tokens[tokenValue]
	if !ok || t.isExpired() {
		return 0, false
	}

	delete(rts.tokens, tokenValue)

	return t.userAccountID, true
}

func (rts *PasswordResetTokenStore) DeleteByUserAccountID(userAccountID int64) {
	rts.Lock()
	defer rts.Unlock()
	for tokenValue, t := range rts.tokens {
		if t.userAccountID == userAccountID {
			delete(rts.tokens, tokenValue)
		}
	}
}
