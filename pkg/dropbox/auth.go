package dropbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	tokenEndpoint    = "https://api.dropboxapi.com/oauth2/token"
	authorizeBaseURL = "https://www.dropbox.com/oauth2/authorize"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type authCodeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	AccountID    string `json:"account_id"`
}

// AuthorizationURL builds the Dropbox OAuth2 authorization URL for the given app key.
// The user opens this URL in a browser, authorizes the app, and receives an authorization code.
func AuthorizationURL(appKey string) string {
	params := url.Values{
		"client_id":         {appKey},
		"response_type":     {"code"},
		"token_access_type": {"offline"},
	}
	return authorizeBaseURL + "?" + params.Encode()
}

// ExchangeAuthorizationCode exchanges an authorization code for a refresh token and access token.
func ExchangeAuthorizationCode(ctx context.Context, appKey, appSecret, code string) (refreshToken, accessToken string, err error) {
	return exchangeAuthorizationCode(ctx, tokenEndpoint, appKey, appSecret, code)
}

func exchangeAuthorizationCode(ctx context.Context, endpoint, appKey, appSecret, code string) (string, string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {appKey},
		"client_secret": {appSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("creating code exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("requesting code exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("code exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tok authCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", "", fmt.Errorf("decoding code exchange response: %w", err)
	}

	if tok.RefreshToken == "" {
		return "", "", fmt.Errorf("empty refresh token in code exchange response")
	}

	if tok.AccessToken == "" {
		return "", "", fmt.Errorf("empty access token in code exchange response")
	}

	return tok.RefreshToken, tok.AccessToken, nil
}

// RefreshAccessToken exchanges a refresh token for a new short-lived access token.
func RefreshAccessToken(ctx context.Context, appKey, appSecret, refreshToken string) (string, error) {
	return refreshAccessToken(ctx, tokenEndpoint, appKey, appSecret, refreshToken)
}

func refreshAccessToken(ctx context.Context, endpoint, appKey, appSecret, refreshToken string) (string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {appKey},
		"client_secret": {appSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting token refresh: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token refresh failed (HTTP %d): %s. Check your app key, app secret, and refresh token", resp.StatusCode, string(body))
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("decoding token refresh response: %w", err)
	}

	if tok.AccessToken == "" {
		return "", fmt.Errorf("empty access token in refresh response")
	}

	return tok.AccessToken, nil
}
