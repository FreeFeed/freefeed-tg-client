package main

//go:generate gotext -srclang=en update -out=catalog_gen.go -lang=en,ru

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/davidmz/debug-log"
	"github.com/davidmz/freefeed-tg-client/app"
	"github.com/davidmz/freefeed-tg-client/store"
	"github.com/davidmz/mustbe"
	tgbotapi "github.com/davidmz/telegram-bot-api"
)

const (
	shutdownTimeout = 10 * time.Second
)

func main() {
	defer mustbe.Catched(func(err error) { log.Fatalln("Fatal error:", err) })

	var (
		tgToken      string
		frfHost      string
		userAgent    string
		dataDir      string
		debugSources string
	)

	flag.StringVar(&tgToken, "token", "", "Telegram bot token (required)")
	flag.StringVar(&frfHost, "host", "freefeed.net", "FreeFeed API/frontend hostname")
	flag.StringVar(&dataDir, "data", "data", "Data directory (must be writable)")
	flag.StringVar(&userAgent, "ua",
		"FreeFeedTelegramClient/1.0 (https://github.com/davidmz/freefeed-tg-client)",
		"User-Agent for backend requests")
	flag.StringVar(&debugSources, "debug", "", "Debug sources, set to '*' to see all messages")
	flag.Parse()

	if tgToken == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "Flags of %s:\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(0)
	}

	if debugSources != "" {
		os.Setenv("DEBUG", debugSources)
	}

	debugLogger := debug.NewLogger("tg-client")
	errorLogger := debug.NewLogger("tg-client:error")
	tgbotapi.SetLogger(debug.NewLogger("tg-client:tgbot"))

	debugLogger.Println("Starting BotAPI")
	tgBot := mustbe.OKVal(tgbotapi.NewBotAPI(tgToken)).(*tgbotapi.BotAPI)

	debugLogger.Printf("Bot authorized on account %s", tgBot.Self.UserName)

	debugLogger.Printf("Starting application")

	a := &app.App{
		DebugLogger:  debugLogger,
		ErrorLogger:  errorLogger,
		Store:        store.NewFsStore(dataDir),
		TgAPI:        tgBot,
		FreeFeedHost: frfHost,
		UserAgent:    userAgent,
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
