// Command natterbot is a ready-to-deploy Telegram bot built on the mtp
// package. It doubles as a worked example of every feature.
//
//	natterbot                     run the bot
//	natterbot secret set KEY      store a secret in the encrypted store
//	natterbot secret list         list stored keys (values masked)
//	natterbot secret rm KEY       delete a stored secret
//	natterbot version             print the version
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nguyenquocanhz/telegramMTP/mtp"
	"github.com/nguyenquocanhz/telegramMTP/secret"
)

// version is overridden at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "natterbot:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "version", "-v", "--version":
			fmt.Println("natterbot", version)
			return nil
		case "secret":
			return secretCLI(args[1:])
		case "help", "-h", "--help":
			fmt.Println(usage)
			return nil
		default:
			return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
		}
	}
	return runBot()
}

const usage = `usage:
  natterbot                  run the bot
  natterbot secret set KEY   store a secret (value is read from stdin)
  natterbot secret list      list stored keys
  natterbot secret rm KEY    delete a stored secret
  natterbot version          print the version

required settings: API_ID, API_HASH, BOT_TOKEN
optional settings: ADMIN_ID (comma separated), SESSION_FILE, LOG_LEVEL`

func runBot() error {
	secrets, err := secret.New(secret.Options{})
	if err != nil {
		return err
	}
	log := newLogger(secrets.GetOr("LOG_LEVEL", "info"))

	apiID, err := secrets.Int64("API_ID")
	if err != nil {
		return err
	}
	apiHash, err := secrets.Require("API_HASH")
	if err != nil {
		return err
	}
	token, err := secrets.Require("BOT_TOKEN")
	if err != nil {
		return err
	}
	admins, err := secrets.Int64List("ADMIN_ID")
	if err != nil {
		return err
	}

	// Log where each credential came from, never the credential itself.
	log.Info("configuration loaded",
		"api_id_from", secrets.Source("API_ID"),
		"api_hash_from", secrets.Source("API_HASH"),
		"bot_token_from", secrets.Source("BOT_TOKEN"),
		"bot_token", secret.Mask(token),
		"admins", admins,
		"version", version,
	)
	if len(admins) == 0 {
		log.Warn("ADMIN_ID is not set: admin commands are disabled. Send /id to the bot to find your ID.")
	}

	bot, err := mtp.New(mtp.Config{
		APIID:       int(apiID),
		APIHash:     apiHash,
		BotToken:    token,
		SessionFile: secrets.GetOr("SESSION_FILE", "data/session.json"),
		AdminIDs:    admins,
		Logger:      log,
	})
	if err != nil {
		return err
	}
	registerHandlers(bot)

	bot.OnStart(func(context.Context) error {
		bot.NotifyAdmins("✅ Bot đã khởi động (" + version + ")")
		return nil
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Info("starting, press Ctrl+C to stop")
	if err := bot.Run(ctx); err != nil {
		return err
	}
	log.Info("stopped")
	return nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
