package mtp

import "slices"

// IsAdmin reports whether userID is listed in Config.AdminIDs.
func (c *Client) IsAdmin(userID int64) bool {
	return slices.Contains(c.cfg.AdminIDs, userID)
}

// AdminIDs returns a copy of the configured admin user IDs.
func (c *Client) AdminIDs() []int64 { return slices.Clone(c.cfg.AdminIDs) }

// OnAdminCommand registers a command only admins may run. For everyone
// else it behaves exactly like an unknown command, and it is listed in
// the command menu of admins only.
func (c *Client) OnAdminCommand(name, description string, h Handler) {
	c.addCommand(&command{name: normalizeCommand(name), desc: description, h: h, admin: true})
}

// NotifyAdmins queues text to every admin. A user can only be messaged
// after they have started the bot, so failures are logged, not returned.
func (c *Client) NotifyAdmins(text string, opts ...SendOption) {
	for _, id := range c.cfg.AdminIDs {
		done := c.QueueMessage(id, text, opts...)
		go func() {
			if err := <-done; err != nil {
				c.log.Warn("notify admin failed (have they sent /start to the bot?)", "admin", id, "err", err)
			}
		}()
	}
}
