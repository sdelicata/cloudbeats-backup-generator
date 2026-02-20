package dropbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

const (
	apiBase        = "https://api.dropboxapi.com/2"
	initialBackoff = 1 * time.Second
	maxBackoff     = 60 * time.Second
)

// Client is a Dropbox API client.
type Client struct {
	token  string
	http   *http.Client
	logger zerolog.Logger
}

// NewClient creates a new Dropbox API client.
func NewClient(token string, logger zerolog.Logger) *Client {
	return &Client{
		token:  token,
		http:   &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

// GetAccountID retrieves the current user's account ID.
func (c *Client) GetAccountID(ctx context.Context) (string, error) {
	body, err := c.apiCall(ctx, "/users/get_current_account", "null")
	if err != nil {
		return "", err
	}
	defer body.Close()

	var account Account
	if err := json.NewDecoder(body).Decode(&account); err != nil {
		return "", fmt.Errorf("failed to decode account response: %w", err)
	}

	if account.AccountID == "" {
		return "", fmt.Errorf("empty account_id in response")
	}

	return account.AccountID, nil
}

// ListFolder lists all file entries under the given remote path (recursive).
// remotePath should be "" for the Dropbox root, not "/".
func (c *Client) ListFolder(ctx context.Context, remotePath string) ([]Entry, error) {
	c.logger.Debug().Str("remote_path", remotePath).Msg("listing Dropbox folder")

	payload := map[string]any{
		"path":      remotePath,
		"recursive": true,
	}
	reqBody, _ := json.Marshal(payload)

	body, err := c.apiCall(ctx, "/files/list_folder", string(reqBody))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp ListFolderResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode list_folder response: %w", err)
	}

	entries := filterFiles(resp.Entries)
	c.logger.Debug().Int("entries", len(entries)).Bool("has_more", resp.HasMore).Msg("received first page")

	for resp.HasMore {
		reqBody, _ := json.Marshal(map[string]string{"cursor": resp.Cursor})

		body, err := c.apiCall(ctx, "/files/list_folder/continue", string(reqBody))
		if err != nil {
			return nil, err
		}

		resp = ListFolderResponse{}
		if err := json.NewDecoder(body).Decode(&resp); err != nil {
			body.Close()
			return nil, fmt.Errorf("failed to decode list_folder/continue response: %w", err)
		}
		body.Close()

		page := filterFiles(resp.Entries)
		entries = append(entries, page...)
		c.logger.Debug().Int("entries", len(page)).Bool("has_more", resp.HasMore).Msg("received continuation page")
	}

	c.logger.Info().Int("total_files", len(entries)).Msg("Dropbox listing complete")
	return entries, nil
}

func filterFiles(entries []Entry) []Entry {
	files := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Tag == "file" {
			files = append(files, e)
		}
	}
	return files
}

func (c *Client) apiCall(ctx context.Context, endpoint, body string) (io.ReadCloser, error) {
	backoff := initialBackoff

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+endpoint, bytes.NewBufferString(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request for %s: %w", endpoint, err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request to %s failed: %w", endpoint, err)
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			return resp.Body, nil

		case resp.StatusCode == http.StatusUnauthorized:
			resp.Body.Close()
			return nil, fmt.Errorf("Dropbox authentication failed (401). " +
				"Your token may be invalid or expired. " +
				"Generate a new token at https://www.dropbox.com/developers/apps")

		case resp.StatusCode == http.StatusTooManyRequests:
			resp.Body.Close()
			wait := backoff
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if secs, err := strconv.Atoi(retryAfter); err == nil {
					wait = time.Duration(secs) * time.Second
				}
			}
			c.logger.Warn().Dur("wait", wait).Msg("rate limited by Dropbox, waiting")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}

			backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))

		default:
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("Dropbox API error %d on %s: %s", resp.StatusCode, endpoint, string(respBody))
		}
	}
}
