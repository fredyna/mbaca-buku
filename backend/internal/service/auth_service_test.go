package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/supabase"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// fakeUserStore records the last password written so tests can assert on the
// stored hash rather than on which methods were called. byEmail is keyed by
// address so OAuth tests can present an account that already exists.
type fakeUserStore struct {
	user        *model.User
	byEmail     map[string]*model.User
	getByIDErr  error
	updatedID   string
	updatedHash string
	updateCalls int

	created   *model.User
	saved     *model.User
	saveCalls int
}

func (f *fakeUserStore) Create(ctx context.Context, user *model.User) error {
	user.ID = "generated-id"
	f.created = user
	return nil
}

func (f *fakeUserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (f *fakeUserStore) Update(ctx context.Context, user *model.User) error {
	f.saveCalls++
	f.saved = user
	return nil
}

func (f *fakeUserStore) GetByID(ctx context.Context, id string) (*model.User, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return f.user, nil
}

func (f *fakeUserStore) UpdatePassword(ctx context.Context, id, hash string) error {
	f.updateCalls++
	f.updatedID = id
	f.updatedHash = hash
	return nil
}

func storeWithPassword(t *testing.T, password string) *fakeUserStore {
	t.Helper()
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	return &fakeUserStore{
		user: &model.User{ID: "user-1", Email: "user@example.com", PasswordHash: hash, Role: "user"},
	}
}

func TestChangePasswordStoresHashOfNewPassword(t *testing.T) {
	store := storeWithPassword(t, "old-password")
	svc := NewAuthService(store, "secret", &fakeVerifier{})

	err := svc.ChangePassword(context.Background(), "user-1", dto.ChangePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "new-password",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-1", store.updatedID)
	assert.True(t, utils.CheckPassword(store.updatedHash, "new-password"),
		"stored hash should verify against the new password")
	assert.False(t, utils.CheckPassword(store.updatedHash, "old-password"),
		"stored hash should no longer verify against the old password")
}

func TestChangePasswordRejectsWrongOldPassword(t *testing.T) {
	store := storeWithPassword(t, "old-password")
	svc := NewAuthService(store, "secret", &fakeVerifier{})

	err := svc.ChangePassword(context.Background(), "user-1", dto.ChangePasswordRequest{
		OldPassword: "wrong-password",
		NewPassword: "new-password",
	})

	assert.ErrorIs(t, err, ErrInvalidOldPassword)
	assert.Zero(t, store.updateCalls, "password must not be written when the old password is wrong")
}

func TestChangePasswordRejectsUnchangedPassword(t *testing.T) {
	store := storeWithPassword(t, "old-password")
	svc := NewAuthService(store, "secret", &fakeVerifier{})

	err := svc.ChangePassword(context.Background(), "user-1", dto.ChangePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "old-password",
	})

	assert.ErrorIs(t, err, ErrSamePassword)
	assert.Zero(t, store.updateCalls, "password must not be written when it is unchanged")
}

func TestChangePasswordPropagatesLookupError(t *testing.T) {
	notFound := errors.New("user not found")
	store := &fakeUserStore{getByIDErr: notFound}
	svc := NewAuthService(store, "secret", &fakeVerifier{})

	err := svc.ChangePassword(context.Background(), "missing", dto.ChangePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "new-password",
	})

	assert.ErrorIs(t, err, notFound)
	assert.Zero(t, store.updateCalls)
}

// fakeVerifier stands in for Supabase. identity is what the token resolves to;
// err short-circuits the lookup.
type fakeVerifier struct {
	identity *supabase.User
	err      error
	gotToken string
	calls    int
}

func (f *fakeVerifier) Configured() bool { return true }

func (f *fakeVerifier) GetUser(ctx context.Context, accessToken string) (*supabase.User, error) {
	f.calls++
	f.gotToken = accessToken
	if f.err != nil {
		return nil, f.err
	}
	return f.identity, nil
}

func googleIdentity() *supabase.User {
	return &supabase.User{
		ID:        "supabase-sub",
		Email:     "gopal@example.com",
		Name:      "Gopal",
		AvatarURL: "https://img/a.png",
		Provider:  "google",
	}
}

func TestOAuthLoginCreatesUserFromVerifiedIdentity(t *testing.T) {
	store := &fakeUserStore{}
	verifier := &fakeVerifier{identity: googleIdentity()}
	svc := NewAuthService(store, "secret", verifier)

	resp, err := svc.OAuthLogin(context.Background(), dto.OAuthRequest{AccessToken: "sb-token"})

	require.NoError(t, err)
	assert.Equal(t, "sb-token", verifier.gotToken, "the token must be redeemed at Supabase")

	require.NotNil(t, store.created)
	assert.Equal(t, "gopal@example.com", store.created.Email)
	assert.Equal(t, "Gopal", store.created.Name)
	assert.Equal(t, "google", store.created.Provider)
	assert.Equal(t, "supabase-sub", store.created.ProviderID)
	assert.Equal(t, "https://img/a.png", store.created.AvatarURL)
	assert.True(t, store.created.IsOAuth)
	assert.Equal(t, "user", store.created.Role, "a new Google account must not arrive as an admin")
	assert.Empty(t, store.created.PasswordHash)

	// The same JWT the password flow issues, so the rest of the API sees no
	// difference between the two ways in.
	userID, role, err := utils.ParseToken(resp.Token, "secret")
	require.NoError(t, err)
	assert.Equal(t, "generated-id", userID)
	assert.Equal(t, "user", role)
	assert.Equal(t, "gopal@example.com", resp.User.Email)
}

func TestOAuthLoginAdoptsExistingAccountByEmailAndKeepsItsRole(t *testing.T) {
	existing := &model.User{
		ID: "admin-1", Name: "Admin", Email: "gopal@example.com",
		PasswordHash: "hash", Role: "admin",
	}
	store := &fakeUserStore{byEmail: map[string]*model.User{"gopal@example.com": existing}}
	svc := NewAuthService(store, "secret", &fakeVerifier{identity: googleIdentity()})

	resp, err := svc.OAuthLogin(context.Background(), dto.OAuthRequest{AccessToken: "sb-token"})

	require.NoError(t, err)
	assert.Nil(t, store.created, "a second account must not be created for a known email")

	require.NotNil(t, store.saved)
	assert.Equal(t, "admin-1", store.saved.ID)
	assert.Equal(t, "admin", store.saved.Role, "the stored role wins over anything the provider says")
	assert.Equal(t, "Gopal", store.saved.Name)
	assert.Equal(t, "supabase-sub", store.saved.ProviderID)
	assert.True(t, store.saved.IsOAuth)
	assert.Equal(t, "hash", store.saved.PasswordHash, "linking must not drop the existing password")

	userID, role, err := utils.ParseToken(resp.Token, "secret")
	require.NoError(t, err)
	assert.Equal(t, "admin-1", userID)
	assert.Equal(t, "admin", role)
}

func TestOAuthLoginSkipsWriteWhenNothingChanged(t *testing.T) {
	existing := &model.User{
		ID: "user-9", Name: "Gopal", Email: "gopal@example.com", Role: "user",
		Provider: "google", ProviderID: "supabase-sub", AvatarURL: "https://img/a.png", IsOAuth: true,
	}
	store := &fakeUserStore{byEmail: map[string]*model.User{"gopal@example.com": existing}}
	svc := NewAuthService(store, "secret", &fakeVerifier{identity: googleIdentity()})

	_, err := svc.OAuthLogin(context.Background(), dto.OAuthRequest{AccessToken: "sb-token"})

	require.NoError(t, err)
	assert.Zero(t, store.saveCalls, "an unchanged profile must not trigger an UPDATE on every sign-in")
}

func TestOAuthLoginRejectsUnverifiableToken(t *testing.T) {
	store := &fakeUserStore{}
	svc := NewAuthService(store, "secret", &fakeVerifier{err: supabase.ErrInvalidToken})

	_, err := svc.OAuthLogin(context.Background(), dto.OAuthRequest{AccessToken: "forged"})

	assert.ErrorIs(t, err, supabase.ErrInvalidToken)
	assert.Nil(t, store.created, "a rejected token must not mint an account")
}

func TestLoginRejectsPasswordAttemptOnGoogleOnlyAccount(t *testing.T) {
	oauthOnly := &model.User{ID: "u1", Email: "gopal@example.com", PasswordHash: "", Role: "user"}
	store := &fakeUserStore{byEmail: map[string]*model.User{"gopal@example.com": oauthOnly}}
	svc := NewAuthService(store, "secret", &fakeVerifier{})

	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "gopal@example.com", Password: "anything",
	})

	assert.ErrorIs(t, err, ErrNoPassword)
}

func TestLoginIssuesTokenForPasswordAccount(t *testing.T) {
	hash, err := utils.HashPassword("s3cret")
	require.NoError(t, err)
	store := &fakeUserStore{byEmail: map[string]*model.User{
		"gopal@example.com": {ID: "u1", Email: "gopal@example.com", PasswordHash: hash, Role: "user"},
	}}
	svc := NewAuthService(store, "secret", &fakeVerifier{})

	resp, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "gopal@example.com", Password: "s3cret",
	})

	require.NoError(t, err)
	userID, role, err := utils.ParseToken(resp.Token, "secret")
	require.NoError(t, err)
	assert.Equal(t, "u1", userID)
	assert.Equal(t, "user", role)
}
