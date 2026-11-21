package mtp

import "testing"

func TestCommandMenus(t *testing.T) {
	c := &Client{commands: make(map[string]*command)}
	noop := func(*Context) error { return nil }
	c.OnCommand("start", "welcome", noop)
	c.OnCommand("secret", "", noop) // no description: works but unlisted
	c.OnAdminCommand("stats", "bot statistics", noop)
	c.OnCommand("help", "list commands", noop)

	names := func(cmds []string) string {
		s := ""
		for _, n := range cmds {
			s += n + " "
		}
		return s
	}
	public, admin := c.commandMenus()
	var pub, adm []string
	for _, b := range public {
		pub = append(pub, b.Command)
	}
	for _, b := range admin {
		adm = append(adm, b.Command)
	}
	if got := names(pub); got != "start help " {
		t.Errorf("public menu = %q", got)
	}
	if got := names(adm); got != "start stats help " {
		t.Errorf("admin menu = %q", got)
	}

	infos := c.Commands()
	if len(infos) != 4 || infos[2].Name != "stats" || !infos[2].AdminOnly || infos[0].AdminOnly {
		t.Errorf("Commands() = %+v", infos)
	}
}
