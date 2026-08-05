package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// fakeUserStore records the last password written so tests can assert on the
// stored hash rather than on which methods were called.
type fakeUserStore struct {
	user        *model.User
	getByIDErr  error
	updatedID   string
	updatedHash string
	updateCalls int
}

func (f *fakeUserStore) Create(ctx context.Context, user *model.User) error {
	return errors.New("not implemented")
}

func (f *fakeUserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, errors.New("not implemented")
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
	svc := NewAuthService(store, "secret")

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
	svc := NewAuthService(store, "secret")

	err := svc.ChangePassword(context.Background(), "user-1", dto.ChangePasswordRequest{
		OldPassword: "wrong-password",
		NewPassword: "new-password",
	})

	assert.ErrorIs(t, err, ErrInvalidOldPassword)
	assert.Zero(t, store.updateCalls, "password must not be written when the old password is wrong")
}

func TestChangePasswordRejectsUnchangedPassword(t *testing.T) {
	store := storeWithPassword(t, "old-password")
	svc := NewAuthService(store, "secret")

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
	svc := NewAuthService(store, "secret")

	err := svc.ChangePassword(context.Background(), "missing", dto.ChangePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "new-password",
	})

	assert.ErrorIs(t, err, notFound)
	assert.Zero(t, store.updateCalls)
}
