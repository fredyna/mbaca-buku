package supabase

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUserReturnsIdentityFromSupabase(t *testing.T) {
	var gotAuth, gotAPIKey, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("apikey")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "9a1f-sub",
			"email": "Fredy@Example.com ",
			"app_metadata": {"provider": "google"},
			"user_metadata": {"full_name": "Fredy Nur", "avatar_url": "https://img/a.png"}
		}`))
	}))
	defer srv.Close()

	user, err := NewVerifier(srv.URL+"/", "anon-key").GetUser(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	if gotPath != "/auth/v1/user" {
		t.Errorf("path = %q, want /auth/v1/user", gotPath)
	}
	if gotAuth != "Bearer token-123" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotAPIKey != "anon-key" {
		t.Errorf("apikey = %q", gotAPIKey)
	}
	// Emails are the account key, so they are lowered and trimmed here rather
	// than trusting the provider to be consistent about casing.
	if user.Email != "fredy@example.com" {
		t.Errorf("Email = %q, want fredy@example.com", user.Email)
	}
	if user.ID != "9a1f-sub" || user.Name != "Fredy Nur" || user.Provider != "google" {
		t.Errorf("unexpected identity: %+v", user)
	}
	if user.AvatarURL != "https://img/a.png" {
		t.Errorf("AvatarURL = %q", user.AvatarURL)
	}
}

func TestGetUserFallsBackToAlternateMetadataKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"id": "sub", "email": "a@b.co",
			"user_metadata": {"name": "Only Name", "picture": "https://img/p.png"}
		}`))
	}))
	defer srv.Close()

	user, err := NewVerifier(srv.URL, "k").GetUser(context.Background(), "t")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.Name != "Only Name" {
		t.Errorf("Name = %q, want Only Name", user.Name)
	}
	if user.AvatarURL != "https://img/p.png" {
		t.Errorf("AvatarURL = %q, want the picture key", user.AvatarURL)
	}
}

func TestGetUserRejectsTokenSupabaseDoesNotRecognise(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := NewVerifier(srv.URL, "k").GetUser(context.Background(), "forged")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestGetUserRejectsEmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("supabase must not be called for an empty token")
	}))
	defer srv.Close()

	_, err := NewVerifier(srv.URL, "k").GetUser(context.Background(), "   ")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestGetUserRequiresEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id": "sub", "email": ""}`))
	}))
	defer srv.Close()

	if _, err := NewVerifier(srv.URL, "k").GetUser(context.Background(), "t"); err == nil {
		t.Fatal("expected an error for a session with no email address")
	}
}

func TestUnconfiguredVerifier(t *testing.T) {
	for name, v := range map[string]*Verifier{
		"no url": NewVerifier("", "k"),
		"no key": NewVerifier("https://x.supabase.co", ""),
	} {
		if v.Configured() {
			t.Errorf("%s: Configured() = true", name)
		}
		if _, err := v.GetUser(context.Background(), "t"); !errors.Is(err, ErrNotConfigured) {
			t.Errorf("%s: err = %v, want ErrNotConfigured", name, err)
		}
	}
}
