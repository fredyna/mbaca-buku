package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// stubUserStore implements service.UserStore with a single known account, so
// these tests exercise the real AuthService rather than a fake of it — the
// point under test is what the handler does after a genuine sign-in succeeds.
type stubUserStore struct {
	user *model.User
}

func (s *stubUserStore) Create(ctx context.Context, user *model.User) error {
	user.ID = "created-id"
	return nil
}

func (s *stubUserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if s.user != nil && s.user.Email == email {
		return s.user, nil
	}
	return nil, errors.New("user not found")
}

func (s *stubUserStore) GetByID(ctx context.Context, id string) (*model.User, error) {
	return s.user, nil
}

func (s *stubUserStore) UpdatePassword(ctx context.Context, id, hash string) error { return nil }

func (s *stubUserStore) Update(ctx context.Context, user *model.User) error { return nil }

type recorderSpy struct {
	calls []recordedCall
}

type recordedCall struct {
	userID string
	event  string
	meta   model.RequestMeta
}

func (s *recorderSpy) RecordAsync(userID, event string, meta model.RequestMeta) {
	s.calls = append(s.calls, recordedCall{userID: userID, event: event, meta: meta})
}

// The recorder parameter is the interface type, not *recorderSpy, so that
// passing nil yields a genuinely nil interface. A nil *recorderSpy would satisfy
// the interface with a non-nil type descriptor, sail past the handler's nil
// check, and panic inside the spy's method.
func loginTestRouter(t *testing.T, recorder activityRecorder) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	hash, err := utils.HashPassword("secret123")
	require.NoError(t, err)

	store := &stubUserStore{user: &model.User{
		ID:           "user-1",
		Name:         "Budi",
		Email:        "budi@example.com",
		PasswordHash: hash,
		Role:         "user",
	}}
	authService := service.NewAuthService(store, "test-secret", nil)
	h := NewAuthHandler(authService, recorder)

	r := gin.New()
	r.POST("/api/auth/login", h.Login)
	return r
}

func postLogin(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLoginRecordsALoginEvent(t *testing.T) {
	spy := &recorderSpy{}
	w := postLogin(loginTestRouter(t, spy), `{"email":"budi@example.com","password":"secret123"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, spy.calls, 1)
	assert.Equal(t, "user-1", spy.calls[0].userID)
	assert.Equal(t, model.EventLogin, spy.calls[0].event)
	assert.Contains(t, spy.calls[0].meta.UserAgent, "Firefox/130")
}

// A failed sign-in belongs in neither the activity log nor a "last login"
// column: it would show an account as active that nobody got into.
func TestLoginRecordsNothingWhenCredentialsAreWrong(t *testing.T) {
	spy := &recorderSpy{}
	w := postLogin(loginTestRouter(t, spy), `{"email":"budi@example.com","password":"wrong-password"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, spy.calls)
}

func TestLoginWorksWithoutARecorder(t *testing.T) {
	w := postLogin(loginTestRouter(t, nil), `{"email":"budi@example.com","password":"secret123"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
