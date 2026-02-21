package dropbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshAccessToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantToken  string
		wantErr    string
	}{
		{
			name:       "successful refresh",
			statusCode: http.StatusOK,
			body:       `{"access_token":"sl.new-token","expires_in":14400,"token_type":"bearer"}`,
			wantToken:  "sl.new-token",
		},
		{
			name:       "invalid credentials",
			statusCode: http.StatusBadRequest,
			body:       `{"error":"invalid_grant"}`,
			wantErr:    "token refresh failed (HTTP 400)",
		},
		{
			name:       "empty access token",
			statusCode: http.StatusOK,
			body:       `{"access_token":"","expires_in":14400,"token_type":"bearer"}`,
			wantErr:    "empty access token in refresh response",
		},
		{
			name:       "invalid JSON response",
			statusCode: http.StatusOK,
			body:       `not json`,
			wantErr:    "decoding token refresh response",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				require.NoError(t, r.ParseForm())
				assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
				assert.Equal(t, "test-refresh", r.FormValue("refresh_token"))
				assert.Equal(t, "test-key", r.FormValue("client_id"))
				assert.Equal(t, "test-secret", r.FormValue("client_secret"))

				w.WriteHeader(test.statusCode)
				_, _ = w.Write([]byte(test.body))
			}))
			defer srv.Close()

			token, err := refreshAccessToken(context.Background(), srv.URL, "test-key", "test-secret", "test-refresh")

			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantToken, token)
		})
	}
}
