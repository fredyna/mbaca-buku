// Package supabase resolves a Supabase access token into the identity it was
// issued for.
//
// The browser finishes the Google OAuth flow against Supabase, not against this
// API, so the only thing it can hand us afterwards is a Supabase access token.
// Everything the API needs about the user — email, name, avatar — also travels
// in that token's payload, but a payload the browser could edit is worth
// nothing: whoever holds the endpoint could mint an app token for any email,
// including the seeded admin. So the token is redeemed at Supabase's own
// /auth/v1/user endpoint and only the answer is trusted.
package supabase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNotConfigured reports that no Supabase project was supplied, so OAuth
// login cannot be served at all.
var ErrNotConfigured = errors.New("supabase is not configured")

// ErrInvalidToken reports that Supabase rejected the access token — it expired,
// was revoked, or never came from this project.
var ErrInvalidToken = errors.New("invalid or expired supabase access token")

// User is the subset of a Supabase user record this API stores.
type User struct {
	// ID is Supabase's own user id (the "sub" claim), stable across logins and
	// across email changes.
	ID    string
	Email string
	// Name and AvatarURL come from the identity provider and may be empty:
	// Google supplies them, other providers need not.
	Name      string
	AvatarURL string
	// Provider names the identity provider behind the session, e.g. "google".
	Provider string
}

type Verifier struct {
	baseURL string
	anonKey string
	client  *http.Client
}

// NewVerifier builds a verifier for a project. An empty baseURL or anonKey
// yields a verifier that fails every call with ErrNotConfigured, which keeps
// OAuth optional without making callers nil-check.
func NewVerifier(baseURL, anonKey string) *Verifier {
	return &Verifier{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		anonKey: anonKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Configured reports whether this verifier can reach a Supabase project.
func (v *Verifier) Configured() bool {
	return v.baseURL != "" && v.anonKey != ""
}

// GetUser asks Supabase who the access token belongs to.
func (v *Verifier) GetUser(ctx context.Context, accessToken string) (*User, error) {
	if !v.Configured() {
		return nil, ErrNotConfigured
	}
	if strings.TrimSpace(accessToken) == "" {
		return nil, ErrInvalidToken
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build supabase request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", v.anonKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach supabase: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, ErrInvalidToken
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("supabase returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		AppMetadata struct {
			Provider string `json:"provider"`
		} `json:"app_metadata"`
		UserMetadata struct {
			FullName  string `json:"full_name"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
			Picture   string `json:"picture"`
		} `json:"user_metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode supabase user: %w", err)
	}

	// A Supabase user can exist without an email — phone sign-in, for one — but
	// this API keys accounts by email, so such a session cannot be honoured.
	if payload.Email == "" {
		return nil, fmt.Errorf("supabase user has no email address")
	}

	return &User{
		ID:        payload.ID,
		Email:     strings.ToLower(strings.TrimSpace(payload.Email)),
		Name:      firstNonEmpty(payload.UserMetadata.FullName, payload.UserMetadata.Name),
		AvatarURL: firstNonEmpty(payload.UserMetadata.AvatarURL, payload.UserMetadata.Picture),
		Provider:  payload.AppMetadata.Provider,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
