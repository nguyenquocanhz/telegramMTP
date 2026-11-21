package mtp

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Handler reacts to one update. A returned error is logged.
type Handler func(c *Context) error

type command struct {
	name string
	desc string
	h    Handler
}

// OnCommand registers a handler for /name. The description is what users
// see in Telegram's command menu. Registering the same name twice
// replaces the earlier handler.
func (c *Client) OnCommand(name, description string, h Handler) {
	c.addCommand(&command{name: normalizeCommand(name), desc: description, h: h})
}

func (c *Client) addCommand(cmd *command) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.commands[cmd.name]; !exists {
		c.order = append(c.order, cmd.name)
	}
	c.commands[cmd.name] = cmd
}

// OnMessage registers a handler for messages that are not a known
// command. Handlers run in registration order.
func (c *Client) OnMessage(h Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onText = append(c.onText, h)
}

func normalizeCommand(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "/"))
}

// parseCommand splits "/name@bot rest of line" into its parts.
func parseCommand(text string) (name, bot, args string, ok bool) {
	if !strings.HasPrefix(text, "/") {
		return "", "", "", false
	}
	head, rest, _ := strings.Cut(text[1:], " ")
	if i := strings.IndexAny(head, "\n\t"); i >= 0 {
		head, rest = head[:i], head[i+1:]+" "+rest
	}
	name, bot, _ = strings.Cut(head, "@")
	if name == "" {
		return "", "", "", false
	}
	return strings.ToLower(name), bot, strings.TrimSpace(rest), true
}

// route picks the handlers for a message: the matching command, or the
// OnMessage handlers when the text is not a command meant for this bot.
func (c *Client) route(x *Context) []Handler {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if name, bot, args, ok := parseCommand(x.Text()); ok {
		// In groups "/cmd@otherbot" is addressed to a different bot.
		if bot != "" && !strings.EqualFold(bot, c.Username()) {
			return nil
		}
		if cmd, found := c.commands[name]; found {
			x.Command, x.Args = name, args
			return []Handler{cmd.h}
		}
	}
	return c.onText
}

func (c *Client) dispatch(x *Context) {
	c.runHandlers(x, c.route(x))
}

func (c *Client) runHandlers(x *Context, handlers []Handler) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("handler panic", "panic", fmt.Sprint(r), "chat", x.chatID, "stack", string(debug.Stack()))
		}
	}()
	for _, h := range handlers {
		if err := h(x); err != nil {
			c.log.Error("handler failed", "err", err, "chat", x.chatID, "command", x.Command)
			return
		}
	}
}
