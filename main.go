package main

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"

	bolt "go.etcd.io/bbolt"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// TODOs
// [x] Enter repos via URL
// [x] Test repos token during creation process
// [ ] Add version

func main() {
	err := godotenv.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Error("Error loading .env file", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("env file loaded")
	}
	appIDFlag := flag.String("app-id", os.Getenv("APP_ID"), "Discord app ID. Can be set by env.")
	botTokenFlag := flag.String("bot-token", os.Getenv("BOT_TOKEN"), "Discord bot token. Can be set by env.")
	logLevelFlag := flag.String("log-level", cmp.Or(os.Getenv("LOG_LEVEL"), "info"), "Set log level for this session. Can be set by env.")
	resetCommandsFlag := flag.Bool("reset-commands", false, "recreates Discord commands. Requires user re-install.")
	flag.Parse()

	// Validations
	if *appIDFlag == "" {
		slog.Error("app ID missing")
		os.Exit(1)
	}
	if *botTokenFlag == "" {
		slog.Error("bot token missing")
		os.Exit(1)
	}

	// set manual log level for this session if requested
	m := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	l, found := m[strings.ToLower(*logLevelFlag)]
	if !found {
		fmt.Println("valid log levels are: ", strings.Join(slices.Collect(maps.Keys(m)), ", "))
		os.Exit(1)
	}
	slog.SetLogLoggerLevel(l)
	slog.SetLogLoggerLevel(slog.LevelInfo)

	db, err := bolt.Open("data.db", 0600, nil)
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

	// Start bot
	ds, err := discordgo.New("Bot " + *botTokenFlag)
	if err != nil {
		slog.Error("Failed to create Discord session", "error", err)
		os.Exit(1)
	}
	ds.Identify.Intents = discordgo.IntentMessageContent
	ds.UserAgent = "SupportBot (https://github.com/ErikKalkoken/discord-supportbot, 0.0.0)"
	b := NewBot(st, ds, *appIDFlag)
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
