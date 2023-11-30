package main

//go:generate gotext -srclang=en update -out=catalog_gen.go -lang=en,ru

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/FreeFeed/freefeed-tg-client/app"
	"github.com/FreeFeed/freefeed-tg-client/store"
	"github.com/davidmz/debug-log"
	"github.com/davidmz/go-try"
	tgbotapi "github.com/davidmz/telegram-bot-api"
)

const (
	shutdownTimeout = 10 * time.Second
)

func main() {
	defer try.Handle(func(err error) { log.Fatalln("Fatal error:", err) })

	var (
		tgToken      string
		tgTokenFile  string
		frfHost      string
		userAgent    string
		dataDir      string
		debugSources string
		noContent    bool
	)

	flag.StringVar(&tgToken, "token", "", "Telegram bot token")
	flag.StringVar(&tgTokenFile, "token-file", "", "Path to the file with Telegram bot token")
	flag.StringVar(&frfHost, "host", "freefeed.net", "FreeFeed API/frontend hostname")
	flag.StringVar(&dataDir, "data", "data", "Data directory (must be writable)")
	flag.StringVar(&userAgent, "ua",
		"FreeFeedTelegramClient/1.0 (https://github.com/FreeFeed/freefeed-tg-client)",
		"User-Agent for backend requests")
	flag.StringVar(&debugSources, "debug", "", "Debug sources, set to '*' to see all messages")
	flag.BoolVar(&noContent, "no-content", false, "Do not include post/comment content into the TG messages")
	flag.Parse()

	if tgToken == "" && tgTokenFile == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "Flags of %s:\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(flag.CommandLine.Output(), "/!\\ Eider -token or -token-file must be specified\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if debugSources != "" {
		os.Setenv("DEBUG", debugSources)
	}

	if tgToken == "" && tgTokenFile != "" {
		tokenData := try.ItVal(os.ReadFile(tgTokenFile))
		tgToken = strings.TrimSpace(string(tokenData))
	}

	debugLogger := debug.NewLogger("tg-client")
	errorLogger := debug.NewLogger("tg-client:error")
	tgbotapi.SetLogger(debug.NewLogger("tg-client:tgbot"))

	debugLogger.Println("Starting BotAPI")
	tgBot, err := tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		try.Throw(fmt.Errorf("cannot start BotAPI: %w", err))
	}

	debugLogger.Printf("Bot authorized on account %s", tgBot.Self.UserName)

	debugLogger.Printf("Starting application")

	a := &app.App{
		DebugLogger:  debugLogger,
		ErrorLogger:  errorLogger,
		Store:        store.NewFsStore(dataDir),
		TgAPI:        tgBot,
		FreeFeedHost: frfHost,
		UserAgent:    userAgent,
		NoContent:    noContent,
	}

	handleStopSignals(a.Close, debugLogger)

	a.Start()

	debugLogger.Println("Bye!")
}

func handleStopSignals(cancel func(), log debug.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		log.Println(s, "signal received, waiting for bot to exit")
		time.AfterFunc(shutdownTimeout, func() {
			log.Println("shutdown timeout, exiting forcefully")
			os.Exit(1)
		})
		cancel()
	}()
}
