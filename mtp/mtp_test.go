package mtp

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestChatIDRoundTrip(t *testing.T) {
	cases := []struct {
		peer   tg.PeerClass
		chatID int64
		kind   peerKind
		id     int64
	}{
		{&tg.PeerUser{UserID: 123456789}, 123456789, kindUser, 123456789},
		{&tg.PeerChat{ChatID: 4567}, -4567, kindChat, 4567},
		{&tg.PeerChannel{ChannelID: 1234567890}, -1001234567890, kindChannel, 1234567890},
	}
	for _, tc := range cases {
		if got := chatIDOf(tc.peer); got != tc.chatID {
			t.Errorf("chatIDOf(%T) = %d, want %d", tc.peer, got, tc.chatID)
		}
		kind, id := splitChatID(tc.chatID)
		if kind != tc.kind || id != tc.id {
			t.Errorf("splitChatID(%d) = (%d, %d), want (%d, %d)", tc.chatID, kind, id, tc.kind, tc.id)
		}
	}
}

func TestParseCommand(t *testing.T) {
	cases := []struct {
		in              string
		name, bot, args string
		ok              bool
	}{
		{"/start", "start", "", "", true},
		{"/Help@NatterBot", "help", "NatterBot", "", true},
		{"/say  hello   world ", "say", "", "hello   world", true},
		{"/echo\nline two", "echo", "", "line two", true},
		{"hello /start", "", "", "", false},
		{"/", "", "", "", false},
		{"", "", "", "", false},
	}
	for _, tc := range cases {
		name, bot, args, ok := parseCommand(tc.in)
		if name != tc.name || bot != tc.bot || args != tc.args || ok != tc.ok {
			t.Errorf("parseCommand(%q) = (%q, %q, %q, %v), want (%q, %q, %q, %v)",
				tc.in, name, bot, args, ok, tc.name, tc.bot, tc.args, tc.ok)
		}
	}
}

func TestRouteCommandAndFallback(t *testing.T) {
	c := &Client{commands: make(map[string]*command)}
	var hit string
	c.OnCommand("/Ping", "health check", func(*Context) error { hit = "ping"; return nil })
	c.OnMessage(func(*Context) error { hit = "text"; return nil })

	run := func(text string) *Context {
		hit = ""
		x := &Context{Client: c, Message: &Message{Text: text}}
		for _, h := range c.route(x) {
			_ = h(x)
		}
		return x
	}

	if x := run("/ping now"); hit != "ping" || x.Command != "ping" || x.Args != "now" {
		t.Errorf("/ping now: hit=%q command=%q args=%q", hit, x.Command, x.Args)
	}
	if run("just talking"); hit != "text" {
		t.Errorf("plain text: hit=%q, want text", hit)
	}
	if run("/unknown"); hit != "text" {
		t.Errorf("unknown command: hit=%q, want text", hit)
	}
	if run("/ping@someoneelse"); hit != "" {
		t.Errorf("command for another bot: hit=%q, want none", hit)
	}
}
