package mtp

import "testing"

func TestAdminCommandIsHiddenFromOthers(t *testing.T) {
	c := &Client{cfg: Config{AdminIDs: []int64{42}}, commands: make(map[string]*command)}
	var hit string
	c.OnAdminCommand("stats", "bot statistics", func(*Context) error { hit = "stats"; return nil })
	c.OnMessage(func(*Context) error { hit = "text"; return nil })

	run := func(sender int64) *Context {
		hit = ""
		x := &Context{Client: c, Message: &Message{Text: "/stats"}, senderID: sender}
		for _, h := range c.route(x) {
			_ = h(x)
		}
		return x
	}

	if x := run(42); hit != "stats" || x.Command != "stats" {
		t.Errorf("admin: hit=%q command=%q", hit, x.Command)
	}
	if x := run(7); hit != "text" || x.Command != "" {
		t.Errorf("non-admin: hit=%q command=%q, want fallthrough to text", hit, x.Command)
	}

	if !c.IsAdmin(42) || c.IsAdmin(7) || c.IsAdmin(0) {
		t.Error("IsAdmin mismatch")
	}
	ids := c.AdminIDs()
	ids[0] = 999
	if !c.IsAdmin(42) {
		t.Error("AdminIDs must return a copy")
	}
}
