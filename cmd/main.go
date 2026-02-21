// Package main is the entry point for the cloudbeats-backup-generator CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/simon/cloudbeats-backup-generator/pkg/backup"
	"github.com/simon/cloudbeats-backup-generator/pkg/dropbox"
	"github.com/simon/cloudbeats-backup-generator/pkg/matcher"
	"github.com/simon/cloudbeats-backup-generator/pkg/tags"
	"github.com/simon/cloudbeats-backup-generator/pkg/worker"
)

func main() {
	localDir := flag.String("local", "", "Path to the local folder to scan (required, must be inside the Dropbox folder)")
	output := flag.String("output", "cloudbeats.cbbackup", "Path to the output .cbbackup file")
	token := flag.String("token", "", "Dropbox access token (also read from DROPBOX_TOKEN env var)")
	workers := flag.Int("workers", 200, "Number of parallel workers for reading tags")
	dryRun := flag.Bool("dry-run", false, "Show Dropbox mapping without reading tags or writing a file")
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

	// Resolve token: flag > env var
	tok := *token
	if tok == "" {
		tok = os.Getenv("DROPBOX_TOKEN")
	}
	if tok == "" {
		logger.Fatal().Msg("Dropbox token is required. Use --token or set DROPBOX_TOKEN env var")
	}

	// Validate workers
	if *workers < 1 {
		logger.Warn().Int("value", *workers).Msg("--workers must be at least 1, clamping to 1")
		*workers = 1
	}

	// Resolve local dir to absolute path
	absLocal, err := filepath.Abs(*localDir)
	if err != nil {
		logger.Fatal().Err(err).Msg("resolving local path")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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

	// Step 3: Read tags with worker pool
	logger.Info().Int("workers", *workers).Msg("reading audio tags...")
	total := len(result.Matched)

	metas, errs := worker.Process(ctx, result.Matched, *workers,
		func(_ context.Context, mf matcher.MatchedFile) (tags.AudioMeta, error) {
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
