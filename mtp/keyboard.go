package mtp

import (
	"strings"

	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
)

// Keyboard is a reply keyboard: buttons shown in place of the user's
// normal keyboard. Tapping a button sends its label as a message, so a
// button labelled "/help" triggers the /help command.
type Keyboard struct {
	rows        [][]string
	oneTime     bool
	persistent  bool
	placeholder string
}

// NewKeyboard builds a keyboard from rows of button labels.
//
//	mtp.NewKeyboard(mtp.KeyRow("/start", "/help"), mtp.KeyRow("/menu"))
func NewKeyboard(rows ...[]string) *Keyboard {
	return &Keyboard{rows: rows}
}

// KeyRow is one row of button labels.
func KeyRow(labels ...string) []string { return labels }

// CommandKeyboard lays commands out perRow buttons to a row, adding the
// leading slash where it is missing.
func CommandKeyboard(perRow int, commands ...string) *Keyboard {
	if perRow < 1 {
		perRow = 2
	}
	var rows [][]string
	for i, name := range commands {
		if i%perRow == 0 {
			rows = append(rows, nil)
		}
		last := len(rows) - 1
		rows[last] = append(rows[last], "/"+strings.TrimPrefix(name, "/"))
	}
	return &Keyboard{rows: rows}
}

// CommandKeyboard builds a keyboard from every command registered with
// OnCommand, in registration order.
func (c *Client) CommandKeyboard(perRow int) *Keyboard {
	c.mu.RLock()
	names := append([]string(nil), c.order...)
	c.mu.RUnlock()
	return CommandKeyboard(perRow, names...)
}

// OneTime hides the keyboard after the first tap.
func (k *Keyboard) OneTime() *Keyboard { k.oneTime = true; return k }

// Persistent keeps the keyboard open instead of collapsing to an icon.
func (k *Keyboard) Persistent() *Keyboard { k.persistent = true; return k }

// Placeholder sets the hint shown in the empty input field.
func (k *Keyboard) Placeholder(text string) *Keyboard { k.placeholder = text; return k }

func (k *Keyboard) markup() tg.ReplyMarkupClass {
	b := markup.BuildKeyboard().Resize()
	if k.oneTime {
		b = b.SingleUse()
	}
	if k.persistent {
		b = b.Persistent()
	}
	if k.placeholder != "" {
		b = b.Placeholder(k.placeholder)
	}
	rows := make([]tg.KeyboardButtonRow, 0, len(k.rows))
	for _, labels := range k.rows {
		buttons := make([]tg.KeyboardButton, 0, len(labels))
		for _, label := range labels {
			buttons = append(buttons, markup.Button(label))
		}
		rows = append(rows, markup.Row(buttons...))
	}
	return b.Build(rows...)
}

// WithKeyboard attaches a reply keyboard to the message.
func WithKeyboard(k *Keyboard) SendOption {
	return func(o *sendOptions) { o.b = o.b.Markup(k.markup()) }
}

// RemoveKeyboard hides a reply keyboard shown by an earlier message.
func RemoveKeyboard() SendOption {
	return func(o *sendOptions) { o.b = o.b.Markup(markup.Hide()) }
}
