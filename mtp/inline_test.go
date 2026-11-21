package mtp

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestInlineMarkup(t *testing.T) {
	m, ok := inlineMarkup([][]InlineButton{
		InlineRow(Btn("Yes", "vote:yes"), Btn("No", "vote:no")),
		InlineRow(URLBtn("Docs", "https://example.com")),
	}).(*tg.ReplyInlineMarkup)
	if !ok {
		t.Fatal("markup is not *tg.ReplyInlineMarkup")
	}
	if len(m.Rows) != 2 || len(m.Rows[0].Buttons) != 2 || len(m.Rows[1].Buttons) != 1 {
		t.Fatalf("unexpected layout: %+v", m.Rows)
	}

	yes := m.Rows[0].Buttons[0]
	cb, ok := yes.Type.(*tg.InlineButtonTypeCallback)
	if !ok || yes.Text != "Yes" || string(cb.Data) != "vote:yes" {
		t.Errorf("callback button = %q %T %+v", yes.Text, yes.Type, yes.Type)
	}
	docs := m.Rows[1].Buttons[0]
	link, ok := docs.Type.(*tg.InlineButtonTypeURL)
	if !ok || link.URL != "https://example.com" {
		t.Errorf("url button = %T %+v", docs.Type, docs.Type)
	}
}

func TestCallbackLongestPrefixWins(t *testing.T) {
	c := &Client{}
	var hit string
	mark := func(name string) Handler {
		return func(*Context) error { hit = name; return nil }
	}
	c.OnCallback("", mark("any"))
	c.OnCallback("menu:", mark("menu"))
	c.OnCallback("menu:photo", mark("photo"))

	for data, want := range map[string]string{
		"menu:photo:1": "photo",
		"menu:about":   "menu",
		"other":        "any",
	} {
		hit = ""
		_ = c.callbackHandler(data)(nil)
		if hit != want {
			t.Errorf("callbackHandler(%q) hit %q, want %q", data, hit, want)
		}
	}

	if (&Client{}).callbackHandler("x") != nil {
		t.Error("no routes registered: want nil handler")
	}
}
