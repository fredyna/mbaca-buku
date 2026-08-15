package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// recorderSpy stands in for *service.ActivityService. It records synchronously,
// which keeps these tests free of sleeps: the middleware calls RecordAsync and
// the spy decides whether that means a goroutine.
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

const testSecret = "test-secret"

func authTestRouter(recorder ActivityRecorder) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/ebooks", AuthMiddleware(testSecret, recorder), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestAuthMiddlewareRecordsActivityForValidToken(t *testing.T) {
	token, err := utils.GenerateToken("user-1", "user", testSecret)
	require.NoError(t, err)

	spy := &recorderSpy{}
	req := httptest.NewRequest(http.MethodGet, "/api/ebooks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	w := httptest.NewRecorder()

	authTestRouter(spy).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, spy.calls, 1)
	assert.Equal(t, "user-1", spy.calls[0].userID)
	assert.Equal(t, model.EventActive, spy.calls[0].event)
	assert.Contains(t, spy.calls[0].meta.UserAgent, "Chrome/128")
	assert.NotEmpty(t, spy.calls[0].meta.IP, "the row is useless without the caller's address")
}

// An unauthenticated request has no user to attribute activity to, and letting
// a rejected caller write rows would make the log forgeable.
func TestAuthMiddlewareRecordsNothingForRejectedRequests(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"no authorization header", ""},
		{"not a bearer token", "Basic dXNlcjpwYXNz"},
		{"signed with the wrong secret", "Bearer " + mustToken(t, "user-1", "other-secret")},
		{"garbage token", "Bearer not-a-jwt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spy := &recorderSpy{}
			req := httptest.NewRequest(http.MethodGet, "/api/ebooks", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()

			authTestRouter(spy).ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Empty(t, spy.calls)
		})
	}
}

// A nil recorder disables logging so tests and any future entry point can build
// the middleware without wiring up a database and Redis.
func TestAuthMiddlewareWorksWithoutARecorder(t *testing.T) {
	token, err := utils.GenerateToken("user-1", "user", testSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/ebooks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		authTestRouter(nil).ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func mustToken(t *testing.T, userID, secret string) string {
	t.Helper()
	token, err := utils.GenerateToken(userID, "user", secret)
	require.NoError(t, err)
	return token
}
