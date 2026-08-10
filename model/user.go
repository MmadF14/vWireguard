package model

import (
	"crypto/subtle"
	"sort"
	"time"
)

// UserRole تعریف نوع نقش کاربر
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleUser    UserRole = "user"
)

// MaxAPITokensPerUser caps how many concurrent sessions a single user can hold.
// When the cap is hit the oldest token is evicted, so a user who never logs out
// cannot grow this list without bound (it lives inside the user's JSON record).
const MaxAPITokensPerUser = 10

// APITokenEntry is one independent, individually-expiring API token.
//
// Previously a user had exactly ONE token field: every new login overwrote it,
// which silently logged out every other device that user had. Tokens are now a
// list, so the site, the desktop client and a phone can each hold their own.
type APITokenEntry struct {
	Token     string    `json:"token"`
	ExpireAt  time.Time `json:"expire_at"`
	CreatedAt time.Time `json:"created_at"`
	// Label is free-form ("site", "windows-client", a User-Agent, ...). Purely
	// informational - useful when showing a user their active sessions.
	Label string `json:"label,omitempty"`
}

// Expired reports whether this entry is past its expiry. A zero ExpireAt means
// "never expires", matching the old TokenExpire semantics.
func (t APITokenEntry) Expired(now time.Time) bool {
	return !t.ExpireAt.IsZero() && now.After(t.ExpireAt)
}

// User model
type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         UserRole  `json:"role"`
	APIToken     string    `json:"api_token,omitempty"`
	TokenExpire  time.Time `json:"token_expire,omitempty"`

	// APITokens holds every currently valid token. APIToken/TokenExpire above
	// are kept only so that records written by an older build keep working and
	// so that any external caller still reading `api_token` sees the most
	// recent one; ValidateAPIToken accepts either representation.
	APITokens []APITokenEntry `json:"api_tokens,omitempty"`
}

// migrateLegacyToken folds a pre-multi-token record into the list exactly once.
func (u *User) migrateLegacyToken() {
	if u.APIToken == "" {
		return
	}
	for _, t := range u.APITokens {
		if t.Token == u.APIToken {
			return
		}
	}
	u.APITokens = append(u.APITokens, APITokenEntry{
		Token:     u.APIToken,
		ExpireAt:  u.TokenExpire,
		CreatedAt: time.Now().UTC(),
		Label:     "legacy",
	})
}

// PruneExpiredTokens drops expired entries. Safe to call at any time.
func (u *User) PruneExpiredTokens() {
	now := time.Now().UTC()
	kept := u.APITokens[:0]
	for _, t := range u.APITokens {
		if !t.Expired(now) {
			kept = append(kept, t)
		}
	}
	u.APITokens = kept
}

// AddAPIToken registers a new token WITHOUT invalidating the existing ones.
func (u *User) AddAPIToken(token string, expireAt time.Time, label string) {
	u.migrateLegacyToken()
	u.PruneExpiredTokens()

	u.APITokens = append(u.APITokens, APITokenEntry{
		Token:     token,
		ExpireAt:  expireAt,
		CreatedAt: time.Now().UTC(),
		Label:     label,
	})

	// Evict oldest first if we are over the cap.
	if len(u.APITokens) > MaxAPITokensPerUser {
		sort.Slice(u.APITokens, func(i, j int) bool {
			return u.APITokens[i].CreatedAt.Before(u.APITokens[j].CreatedAt)
		})
		u.APITokens = u.APITokens[len(u.APITokens)-MaxAPITokensPerUser:]
	}

	// Mirror the newest token into the legacy fields for backward compatibility.
	u.APIToken = token
	u.TokenExpire = expireAt
}

// ValidateAPIToken reports whether token is currently valid for this user.
// Comparison is constant-time so a token cannot be recovered by timing.
func (u *User) ValidateAPIToken(token string) bool {
	if token == "" {
		return false
	}
	now := time.Now().UTC()

	for _, t := range u.APITokens {
		if subtle.ConstantTimeCompare([]byte(t.Token), []byte(token)) == 1 {
			return !t.Expired(now)
		}
	}

	// Fall back to a record that predates the list.
	if u.APIToken != "" && subtle.ConstantTimeCompare([]byte(u.APIToken), []byte(token)) == 1 {
		return u.TokenExpire.IsZero() || !now.After(u.TokenExpire)
	}
	return false
}

// RevokeAPIToken removes a single token (logout on one device only).
// Reports whether anything was removed.
func (u *User) RevokeAPIToken(token string) bool {
	removed := false
	kept := u.APITokens[:0]
	for _, t := range u.APITokens {
		if t.Token == token {
			removed = true
			continue
		}
		kept = append(kept, t)
	}
	u.APITokens = kept

	if u.APIToken == token {
		u.APIToken = ""
		u.TokenExpire = time.Time{}
		removed = true
		// Keep the legacy mirror pointing at whatever is still valid.
		if len(u.APITokens) > 0 {
			newest := u.APITokens[0]
			for _, t := range u.APITokens[1:] {
				if t.CreatedAt.After(newest.CreatedAt) {
					newest = t
				}
			}
			u.APIToken = newest.Token
			u.TokenExpire = newest.ExpireAt
		}
	}
	return removed
}

// RevokeAllAPITokens logs the user out everywhere.
func (u *User) RevokeAllAPITokens() {
	u.APITokens = nil
	u.APIToken = ""
	u.TokenExpire = time.Time{}
}
