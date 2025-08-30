package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const (
	name    = "issuebot"
	dbName  = name + ".db"
	repoURL = "https://github.com/ErikKalkoken/discord-issuebot"
)

// Version - needs to be injected via ldflags.
var Version = "0.0.0"

func main() {
	appIDFlag := flag.String("app-id", "", "Discord app ID. Can be set by env.")
	botTokenFlag := flag.String("bot-token", "", "Discord bot token. Can be set by env.")
	logLevelFlag := flag.String("log-level", "info", "Set log level for this session. Can be set by env.")
	resetCommandsFlag := flag.Bool("reset-commands", false, "Recreates Discord commands. Requires user re-install.")
	versionFlag := flag.Bool("version", false, "Shows the version.")
	exportFlag := flag.Bool("export", false, "export data as JSON")
	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}
	err := godotenv.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Error("Error loading .env file", "error", err)
			os.Exit(1)
		}
	}

	// Validations
	appID := cmp.Or(*appIDFlag, os.Getenv("APP_ID"))
	if appID == "" {
		slog.Error("app ID missing")
		os.Exit(1)
	}
	botToken := cmp.Or(*botTokenFlag, os.Getenv("BOT_TOKEN"))
	if botToken == "" {
		slog.Error("bot token missing")
		os.Exit(1)
	}

	// set manual log level for this session if requested
	logLevel := cmp.Or(*logLevelFlag, os.Getenv("LOG_LEVEL"))
	m := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	l, found := m[strings.ToLower(logLevel)]
	if !found {
		fmt.Println("valid log levels are: ", strings.Join(slices.Collect(maps.Keys(m)), ", "))
		os.Exit(1)
	}
	slog.SetLogLoggerLevel(l)

	db, err := bolt.Open(dbName, 0600, nil)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	st := NewStorage(db)
	if err := st.Init(); err != nil {
		slog.Error("Failed to init database", "error", err)
		os.Exit(1)
	}

	if *exportFlag {
		data, err := func() ([]byte, error) {
			repos, err := st.ListAllRepos()
			if err != nil {
				return nil, err
			}
			if len(repos) > 0 {
				slices.SortFunc(repos, func(a, b *Repo) int {
					return cmp.Compare(a.ID, b.ID)
				})
			}
			data, err := json.MarshalIndent(repos, "", "    ")
			if err != nil {
				return nil, err
			}
			return data, nil
		}()
		if err != nil {
			slog.Error("Failed to list all repos", "error", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		os.Exit(0)
	}

	// Start bot
	ds, err := discordgo.New("Bot " + botToken)
	if err != nil {
		slog.Error("Failed to create Discord session", "error", err)
		os.Exit(1)
	}
	ds.Identify.Intents = discordgo.IntentMessageContent
	ds.UserAgent = fmt.Sprintf("%s (%s, %s)", name, repoURL, Version)

	api := newRepoAPI()
	api.HTTPClient = &http.Client{
		Timeout: time.Second * 5,
	}

	b := NewBot(st, ds, appID, api)
	if err := ds.Open(); err != nil {
		slog.Error("Cannot open the Discord session", "error", err)
		os.Exit(1)
	}
	defer ds.Close()
	if err := b.InitCommands(*resetCommandsFlag); err != nil {
		slog.Error("Failed to init Discord commands", "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	slog.Info("Graceful shutdown")
}
