// Package mtp is a small, Telethon-flavoured wrapper around gotd/td.
//
// It talks MTProto directly (no Bot API HTTP gateway) but hides the raw
// TL types behind a handful of calls:
//
//	bot, _ := mtp.New(mtp.Config{APIID: id, APIHash: hash, BotToken: token})
//	bot.OnCommand("ping", "health check", func(c *mtp.Context) error {
//		_, err := c.Reply("pong")
//		return err
//	})
//	bot.Run(ctx)
//
// Chats are addressed by a single int64 using the same convention as
// Telethon "marked" IDs and the Bot API: users are positive, basic groups
// are negative, channels/supergroups are -100<id>.
package mtp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
)

// Config holds everything needed to connect.
type Config struct {
	// APIID and APIHash come from https://my.telegram.org/apps.
	APIID   int
	APIHash string
	// BotToken comes from @BotFather.
	BotToken string
	// SessionFile stores the MTProto auth key so restarts do not log in
	// again. Treat it like a password. Defaults to "session.json".
	SessionFile string
	// Logger defaults to slog.Default().
	Logger *slog.Logger
}

// Client is a connected (or connectable) Telegram bot.
type Client struct {
	cfg    Config
	log    *slog.Logger
	tg     *telegram.Client
	api    *tg.Client
	sender *message.Sender
	peers  *peerCache
	self   atomic.Pointer[tg.User]

	mu        sync.RWMutex
	commands  map[string]*command
	order     []string // command names in registration order
	onText    []Handler
	callbacks []callbackRoute
	onStart   []func(ctx context.Context) error
}

// New validates cfg and prepares a client. Nothing touches the network
// until Run is called.
func New(cfg Config) (*Client, error) {
	switch {
	case cfg.APIID == 0:
		return nil, errors.New("mtp: APIID is required")
	case cfg.APIHash == "":
		return nil, errors.New("mtp: APIHash is required")
	case cfg.BotToken == "":
		return nil, errors.New("mtp: BotToken is required")
	}
	if cfg.SessionFile == "" {
		cfg.SessionFile = "session.json"
	}
	if dir := filepath.Dir(cfg.SessionFile); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("mtp: session dir: %w", err)
		}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	c := &Client{
		cfg:      cfg,
		log:      cfg.Logger,
		peers:    newPeerCache(),
		commands: make(map[string]*command),
	}

	d := tg.NewUpdateDispatcher()
	d.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		c.handleMessage(ctx, e, u.Message)
		return nil
	})
	d.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		c.handleMessage(ctx, e, u.Message)
		return nil
	})
	d.OnBotCallbackQuery(func(ctx context.Context, e tg.Entities, u *tg.UpdateBotCallbackQuery) error {
		c.handleCallback(ctx, e, u)
		return nil
	})

	c.tg = telegram.NewClient(cfg.APIID, cfg.APIHash, telegram.Options{
		UpdateHandler:  d,
		SessionStorage: &session.FileStorage{Path: cfg.SessionFile},
	})
	c.api = c.tg.API()
	c.sender = message.NewSender(c.api)
	return c, nil
}

// OnStart registers fn to run once the bot is logged in, before Run
// starts waiting. A returned error stops the bot.
func (c *Client) OnStart(fn func(ctx context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStart = append(c.onStart, fn)
}

// Run connects, logs in with the bot token (only on first start; later
// starts reuse the session file) and blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) error {
	err := c.tg.Run(ctx, func(ctx context.Context) error {
		status, err := c.tg.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("auth status: %w", err)
		}
		if !status.Authorized {
			if _, err := c.tg.Auth().Bot(ctx, c.cfg.BotToken); err != nil {
				return fmt.Errorf("bot login: %w", err)
			}
		}
		self, err := c.tg.Self(ctx)
		if err != nil {
			return fmt.Errorf("get self: %w", err)
		}
		c.self.Store(self)
		c.log.Info("connected", "username", self.Username, "id", self.ID)

		c.mu.RLock()
		hooks := append([]func(context.Context) error(nil), c.onStart...)
		c.mu.RUnlock()
		for _, hook := range hooks {
			if err := hook(ctx); err != nil {
				return err
			}
		}

		<-ctx.Done()
		return ctx.Err()
	})
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// Username returns the bot's own username, or "" before Run has logged in.
func (c *Client) Username() string {
	if self := c.self.Load(); self != nil {
		return self.Username
	}
	return ""
}

// API exposes the raw gotd client for calls this package does not wrap.
func (c *Client) API() *tg.Client { return c.api }

func (c *Client) handleMessage(ctx context.Context, e tg.Entities, mc tg.MessageClass) {
	m, ok := mc.(*tg.Message)
	if !ok || m.Out {
		return
	}
	c.peers.learn(e)
	msg := newMessage(m)
	x := &Context{
		Context:  ctx,
		Client:   c,
		Message:  msg,
		chatID:   msg.ChatID,
		senderID: msg.SenderID,
		msgID:    msg.ID,
	}
	// Each update gets its own goroutine so a slow handler never blocks
	// the update stream.
	go c.dispatch(x)
}
