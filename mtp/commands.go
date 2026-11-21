package mtp

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// CommandInfo describes a registered command.
type CommandInfo struct {
	Name        string
	Description string
	AdminOnly   bool
}

// Commands lists the registered commands in registration order.
func (c *Client) Commands() []CommandInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]CommandInfo, 0, len(c.order))
	for _, name := range c.order {
		cmd := c.commands[name]
		out = append(out, CommandInfo{Name: cmd.name, Description: cmd.desc, AdminOnly: cmd.admin})
	}
	return out
}

// commandMenus builds the two menus Telegram shows when a user types
// "/": the public one, and public+admin for admins. Commands without a
// description stay unlisted but still work.
func (c *Client) commandMenus() (public, admin []tg.BotCommand) {
	for _, info := range c.Commands() {
		if info.Description == "" {
			continue
		}
		bc := tg.BotCommand{Command: info.Name, Description: info.Description}
		admin = append(admin, bc)
		if !info.AdminOnly {
			public = append(public, bc)
		}
	}
	return public, admin
}

// SyncCommands publishes the command list to Telegram so it appears in
// the chat's "/" menu. Run calls it on every start; call it again after
// registering commands at runtime.
func (c *Client) SyncCommands(ctx context.Context) error {
	public, admin := c.commandMenus()

	if _, err := c.api.BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
		Scope:    &tg.BotCommandScopeDefault{},
		Commands: public,
	}); err != nil {
		return fmt.Errorf("mtp: set commands: %w", err)
	}
	if len(admin) == len(public) {
		return nil // no admin-only commands to add
	}

	for _, id := range c.cfg.AdminIDs {
		peer, err := c.resolve(ctx, id)
		if err == nil {
			_, err = c.api.BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
				Scope:    &tg.BotCommandScopePeer{Peer: peer},
				Commands: admin,
			})
		}
		if err != nil {
			// Usually the admin has not opened a chat with the bot yet.
			// The commands still work; only the menu entry is missing.
			c.log.Warn("could not set admin command menu", "admin", id, "err", err)
		}
	}
	return nil
}
