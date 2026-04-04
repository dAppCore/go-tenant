// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"
	"time"
)

// User is the authenticated identity. Tier controls feature access for personal namespaces.
//
//	u, err := tenant.UserFromCtx(ctx)
//	if u.Tier == tenant.TierHades { ... }
type User struct {
	ID              int64      `json:"id"`
	UUID            string     `json:"uuid"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Tier            UserTier   `json:"tier"`
	TierExpiresAt   *time.Time `json:"tier_expires_at"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// UserTier maps to PHP's UserTier enum.
type UserTier string

const (
	TierFree   UserTier = "free"
	TierApollo UserTier = "apollo" // Standard paid tier
	TierHades  UserTier = "hades"  // Premium tier
)

// MaxWorkspaces returns the workspace limit for this tier.
// Returns -1 for unlimited (Hades).
//
//	if u.Tier.MaxWorkspaces() != -1 && count >= u.Tier.MaxWorkspaces() { ... }
func (t UserTier) MaxWorkspaces() int {
	switch t {
	case TierApollo:
		return 5
	case TierHades:
		return -1
	default:
		return 1
	}
}

// HasFeature checks whether this tier includes the named feature code.
//
//	if u.Tier.HasFeature("api_access") { ... }
func (t UserTier) HasFeature(code string) bool {
	// TODO: implement — tier-to-feature mapping
	return false
}

// UserFromCtx retrieves the authenticated user from context.
// Returns ErrNoUserContext if no user was injected.
//
//	user, err := tenant.UserFromCtx(ctx)
func UserFromCtx(ctx context.Context) (*User, error) {
	// TODO: implement
	return nil, ErrNoUserContext
}

// WithUser returns a new context carrying the user.
//
//	ctx = tenant.WithUser(ctx, user)
func WithUser(ctx context.Context, user *User) context.Context {
	// TODO: implement
	return ctx
}
