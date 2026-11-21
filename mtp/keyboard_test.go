package mtp

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestCommandKeyboardLayout(t *testing.T) {
	k := CommandKeyboard(2, "start", "/help", "menu").OneTime().Placeholder("pick one")

	m, ok := k.markup().(*tg.ReplyKeyboardMarkup)
	if !ok {
		t.Fatalf("markup is %T, want *tg.ReplyKeyboardMarkup", k.markup())
	}
	if !m.Resize || !m.SingleUse || m.Placeholder != "pick one" {
		t.Errorf("flags: resize=%v singleUse=%v placeholder=%q", m.Resize, m.SingleUse, m.Placeholder)
	}

	want := [][]string{{"/start", "/help"}, {"/menu"}}
	if len(m.Rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(m.Rows), len(want))
	}
	for i, row := range m.Rows {
		if len(row.Buttons) != len(want[i]) {
			t.Fatalf("row %d: got %d buttons, want %d", i, len(row.Buttons), len(want[i]))
		}
		for j, b := range row.Buttons {
			if b.Text != want[i][j] {
				t.Errorf("row %d button %d = %q, want %q", i, j, b.Text, want[i][j])
			}
		}
	}
}

func TestClientCommandKeyboardUsesRegistrationOrder(t *testing.T) {
	c := &Client{commands: make(map[string]*command)}
	noop := func(*Context) error { return nil }
	c.OnCommand("start", "", noop)
	c.OnCommand("help", "", noop)
	c.OnCommand("start", "again", noop) // re-registering must not duplicate

	m := c.CommandKeyboard(3).markup().(*tg.ReplyKeyboardMarkup)
	if len(m.Rows) != 1 || len(m.Rows[0].Buttons) != 2 {
		t.Fatalf("unexpected layout: %+v", m.Rows)
	}
	if m.Rows[0].Buttons[0].Text != "/start" || m.Rows[0].Buttons[1].Text != "/help" {
		t.Errorf("order = %q, %q", m.Rows[0].Buttons[0].Text, m.Rows[0].Buttons[1].Text)
	}
}
