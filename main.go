package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

func main() {
	var (
		resetCommandsFlag = flag.Bool("reset-commands", false, "recreates Discord commands. Requires user re-install.")
	)
	flag.Parse()

	config, err := readConfig("config.yaml")
	if err != nil {
		slog.Error("Configuration error", "error", err)
		os.Exit(1)
	}

	// fmt.Printf("%+v", config)
	// return

	// Start bot
	ds, err := discordgo.New("Bot " + config.Discord.BotToken)
	if err != nil {
		slog.Error("Failed to create Discord session", "error", err)
		os.Exit(1)
	}
	ds.Identify.Intents = discordgo.IntentMessageContent
	ds.UserAgent = "SupportBot (https://github.com/ErikKalkoken/discord-supportbot, 0.0.0)"
	b := NewBot(ds, config)
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
