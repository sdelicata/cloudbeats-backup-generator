// Package main is the entry point for the cloudbeats-backup-generator CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/rs/zerolog"

	"github.com/sdelicata/cloudbeats-backup-generator/pkg/backup"
	"github.com/sdelicata/cloudbeats-backup-generator/pkg/cache"
	"github.com/sdelicata/cloudbeats-backup-generator/pkg/config"
	"github.com/sdelicata/cloudbeats-backup-generator/pkg/dropbox"
	"github.com/sdelicata/cloudbeats-backup-generator/pkg/matcher"
	"github.com/sdelicata/cloudbeats-backup-generator/pkg/tags"
	"github.com/sdelicata/cloudbeats-backup-generator/pkg/worker"
)

func main() {
	localDir := flag.String("local", "", "Path to the local folder to scan (required, must be inside the Dropbox folder)")
	output := flag.String("output", "cloudbeats.cbbackup", "Path to the output .cbbackup file")
	token := flag.String("token", "", "Dropbox access token (also read from DROPBOX_TOKEN env var)")
	appKey := flag.String("app-key", "", "Dropbox app key for refresh token auth (also read from DROPBOX_APP_KEY env var)")
	appSecret := flag.String("app-secret", "", "Dropbox app secret for refresh token auth (also read from DROPBOX_APP_SECRET env var)")
	refreshToken := flag.String("refresh-token", "", "Dropbox refresh token for automatic token renewal (also read from DROPBOX_REFRESH_TOKEN env var)")
	workers := flag.Int("workers", 0, "Number of parallel workers for reading tags (0 = auto: 2x CPU cores)")
	dryRun := flag.Bool("dry-run", false, "Show Dropbox mapping without reading tags or writing a file")
	noCache := flag.Bool("no-cache", false, "Disable the tag cache (re-parse all files)")
	logLevel := flag.String("log-level", "info", "Log level: trace, debug, info, warn, error")
	flag.Parse()

	// Setup logger
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Timestamp().Logger().
		Level(level)

	// Validate required flags
	if *localDir == "" {
		logger.Fatal().Msg("--local flag is required")
	}

	// Resolve Dropbox access token
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ak := firstNonEmpty(*appKey, os.Getenv("DROPBOX_APP_KEY"))
	as := firstNonEmpty(*appSecret, os.Getenv("DROPBOX_APP_SECRET"))
	rt := firstNonEmpty(*refreshToken, os.Getenv("DROPBOX_REFRESH_TOKEN"))
	dt := firstNonEmpty(*token, os.Getenv("DROPBOX_TOKEN"))

	tok, err := resolveToken(ctx, ak, as, rt, dt, logger)
	if err != nil {
		if !isInteractive() {
			logger.Fatal().Err(err).Msg("resolving Dropbox token")
		}

		// Interactive auto-setup
		logger.Warn().Msg("no Dropbox credentials found, starting interactive setup...")
		if ak == "" {
			ak = promptValue("Dropbox app key")
		}
		if as == "" {
			as = promptValue("Dropbox app secret")
		}
		if err := runAuth(ctx, ak, as, logger); err != nil {
			logger.Fatal().Err(err).Msg("authorization failed")
		}

		// Retry with saved credentials
		tok, err = resolveToken(ctx, "", "", "", "", logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("resolving Dropbox token after setup")
		}
	}

	// Auto-detect or validate workers
	if *workers <= 0 {
		*workers = runtime.NumCPU() * 2
	}

	// Resolve local dir to absolute path
	absLocal, err := filepath.Abs(*localDir)
	if err != nil {
		logger.Fatal().Err(err).Msg("resolving local path")
	}

	// Step 1: Authenticate with Dropbox
	client := dropbox.NewClient(tok, logger)
	logger.Info().Msg("authenticating with Dropbox...")
	accountID, err := client.GetAccountID(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("authenticating with Dropbox")
	}
	logger.Info().Str("account_id", accountID).Msg("authenticated")

	// Step 2a: Detect Dropbox root path
	dropboxRoot, err := dropbox.DetectRootPath()
	if err != nil {
		logger.Fatal().Err(err).Msg("detecting Dropbox root path")
	}
	logger.Info().Str("dropbox_root", dropboxRoot).Msg("detected Dropbox root")

	// Step 2b: Compute remote path
	remotePath, err := dropbox.ComputeRemotePath(absLocal, dropboxRoot)
	if err != nil {
		logger.Fatal().Err(err).Msg("computing remote path")
	}
	logger.Info().Str("remote_path", remotePath).Msg("computed remote path")

	// Step 2c: Scan local files
	logger.Info().Str("dir", absLocal).Msg("scanning local files...")
	localFiles, err := matcher.ScanLocal(absLocal)
	if err != nil {
		logger.Fatal().Err(err).Msg("scanning local directory")
	}
	logger.Info().Int("count", len(localFiles)).Msg("local audio files found")

	// Step 2d: List Dropbox files
	logger.Info().Msg("listing Dropbox files...")
	entries, err := client.ListFolder(ctx, remotePath)
	if err != nil {
		logger.Fatal().Err(err).Msg("listing Dropbox folder")
	}

	// Step 2e: Match local files with Dropbox entries
	result := matcher.Match(absLocal, remotePath, localFiles, entries)
	logger.Info().
		Int("matched", len(result.Matched)).
		Int("unmatched_local", len(result.UnmatchedLocal)).
		Int("unmatched_dropbox", len(result.UnmatchedDropbox)).
		Msg("matching complete")

	// Log unmatched files
	for _, path := range result.UnmatchedLocal {
		logger.Debug().Str("file", path).Msg("local file has no Dropbox match (skipped)")
	}
	for _, entry := range result.UnmatchedDropbox {
		logger.Debug().Str("path", entry.PathDisplay).Msg("Dropbox file has no local match")
	}

	// Dry-run: print summary and exit
	if *dryRun {
		fmt.Fprintf(os.Stderr, "\n--- Dry Run Summary ---\n")
		fmt.Fprintf(os.Stderr, "Remote path:       %s\n", remotePath)
		fmt.Fprintf(os.Stderr, "Local files:       %d\n", len(localFiles))
		fmt.Fprintf(os.Stderr, "Dropbox files:     %d\n", len(entries))
		fmt.Fprintf(os.Stderr, "Matched:           %d\n", len(result.Matched))
		fmt.Fprintf(os.Stderr, "Unmatched local:   %d\n", len(result.UnmatchedLocal))
		fmt.Fprintf(os.Stderr, "Unmatched Dropbox: %d\n", len(result.UnmatchedDropbox))
		return
	}

	// Load tag cache
	var tagCache *cache.TagCache
	if !*noCache {
		tagCache = cache.Load(defaultCachePath(), logger)
		logger.Info().Int("entries", tagCache.Len()).Msg("tag cache loaded")
	}

	// Step 3: Read tags with worker pool
	logger.Info().Int("workers", *workers).Msg("reading audio tags...")
	total := len(result.Matched)

	var cacheHits atomic.Int64
	metas, errs := worker.Process(ctx, result.Matched, *workers,
		func(_ context.Context, mf matcher.MatchedFile) (tags.AudioMeta, error) {
			if tagCache != nil {
				if meta, ok := tagCache.Lookup(mf.LocalPath); ok {
					cacheHits.Add(1)
					return meta, nil
				}
			}
			return tags.ReadFile(mf.LocalPath)
		},
		func(done, total int) {
			fmt.Fprintf(os.Stderr, "\rProcessing: %d/%d files", done, total)
		},
	)
	fmt.Fprintf(os.Stderr, "\rProcessing: %d/%d files\n", total, total)

	// Log any tag reading errors (e.g. taglib panics)
	for i, err := range errs {
		if err != nil {
			logger.Warn().Err(err).Str("file", result.Matched[i].LocalPath).Msg("error reading tags")
		}
	}

	// Update and save tag cache
	if tagCache != nil {
		for i, mf := range result.Matched {
			if errs[i] == nil {
				tagCache.Store(mf.LocalPath, metas[i])
			}
		}
		if err := tagCache.Save(); err != nil {
			logger.Warn().Err(err).Msg("saving tag cache")
		}
		logger.Info().
			Int("hits", int(cacheHits.Load())).
			Int("parsed", total-int(cacheHits.Load())).
			Msg("tag cache stats")
	}

	// Step 4: Build backup items
	items := make([]backup.Item, len(result.Matched))
	for i, mf := range result.Matched {
		meta := metas[i]
		item := backup.Item{
			AccountID:   accountID,
			Key:         mf.Entry.ID,
			Name:        mf.Entry.Name,
			Path:        "",
			Service:     "dropbox",
			Album:       meta.Album,
			AlbumArtist: meta.AlbumArtist,
			Artist:      meta.Artist,
			DiskNumber:  meta.DiskNumber,
			Duration:    backup.Duration(meta.Duration.Seconds()),
			TagName:     meta.Title,
			Year:        meta.Year,
		}
		if meta.Genre != "" {
			item.Genre = &meta.Genre
		}
		if meta.TrackNumber >= 0 {
			item.TrackNumber = &meta.TrackNumber
		}
		items[i] = item
	}

	b := &backup.Backup{
		Items:     items,
		Playlists: []backup.Playlist{},
	}

	// Step 5: Write backup file
	if err := backup.Write(*output, b); err != nil {
		logger.Fatal().Err(err).Msg("writing backup file")
	}
	logger.Info().Str("output", *output).Int("items", len(items)).Msg("backup file written")
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func promptValue(name string) string {
	fmt.Fprintf(os.Stderr, "%s: ", name)
	var value string
	if _, err := fmt.Scanln(&value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func runAuth(ctx context.Context, appKey, appSecret string, logger zerolog.Logger) error {
	authURL := dropbox.AuthorizationURL(appKey)
	fmt.Fprintf(os.Stderr, "Opening authorization URL in your browser...\n\n  %s\n\n", authURL)
	openBrowser(authURL)

	fmt.Fprint(os.Stderr, "Paste the authorization code here: ")
	var code string
	if _, err := fmt.Scanln(&code); err != nil {
		return fmt.Errorf("reading authorization code: %w", err)
	}
	code = strings.TrimSpace(code)

	if code == "" {
		return fmt.Errorf("authorization code cannot be empty")
	}

	logger.Info().Msg("exchanging authorization code...")
	refreshToken, _, err := dropbox.ExchangeAuthorizationCode(ctx, appKey, appSecret, code)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	creds := &config.Credentials{
		AppKey:       appKey,
		AppSecret:    appSecret,
		RefreshToken: refreshToken,
	}
	if err := config.Save(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Credentials saved. You can now run the tool without any auth flags.\n")
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}

func resolveToken(ctx context.Context, appKey, appSecret, refreshToken, directToken string, logger zerolog.Logger) (string, error) {
	// Explicit flags: all 3 refresh params present
	if appKey != "" && appSecret != "" && refreshToken != "" {
		logger.Info().Msg("refreshing Dropbox access token...")
		token, err := dropbox.RefreshAccessToken(ctx, appKey, appSecret, refreshToken)
		if err != nil {
			return "", fmt.Errorf("refreshing access token: %w", err)
		}
		logger.Info().Msg("access token refreshed successfully")
		return token, nil
	}

	// Stored credentials
	creds, err := config.Load()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to load stored credentials")
	}
	if creds != nil && creds.AppKey != "" && creds.AppSecret != "" && creds.RefreshToken != "" {
		logger.Info().Msg("using stored credentials, refreshing access token...")
		token, err := dropbox.RefreshAccessToken(ctx, creds.AppKey, creds.AppSecret, creds.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("refreshing access token with stored credentials: %w", err)
		}
		logger.Info().Msg("access token refreshed successfully")
		return token, nil
	}

	// Direct mode: token provided directly
	if directToken != "" {
		return directToken, nil
	}

	return "", fmt.Errorf("dropbox authentication required. Either:\n" +
		"  - Provide --app-key, --app-secret, and --refresh-token\n" +
		"  - Provide --token or DROPBOX_TOKEN env var (short-lived, expires in ~4h)\n" +
		"  - Run interactively to set up credentials (one-time setup)")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func defaultCachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "cloudbeats-backup-generator", "cache.json")
}
