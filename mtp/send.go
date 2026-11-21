package mtp

import (
	"context"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/message/unpack"
)

// SendOption tweaks an outgoing message.
type SendOption func(*sendOptions)

type sendOptions struct {
	b    *message.Builder
	html bool
}

// Silent sends without a notification sound.
func Silent() SendOption {
	return func(o *sendOptions) { o.b = o.b.Silent() }
}

// NoPreview disables the link preview.
func NoPreview() SendOption {
	return func(o *sendOptions) { o.b = o.b.NoWebpage() }
}

// HTML parses the text as Telegram HTML (<b>, <i>, <code>, <a href>, ...).
func HTML() SendOption {
	return func(o *sendOptions) { o.html = true }
}

// prepare resolves the chat and applies the options to a fresh builder.
func (c *Client) prepare(ctx context.Context, chatID int64, opts []SendOption) (*sendOptions, error) {
	peer, err := c.resolve(ctx, chatID)
	if err != nil {
		return nil, err
	}
	o := &sendOptions{b: c.sender.To(peer).CloneBuilder()}
	for _, opt := range opts {
		opt(o)
	}
	return o, nil
}

// styled converts text into gotd styling according to the options.
func (o *sendOptions) styled(text string) []message.StyledTextOption {
	if o.html {
		return []message.StyledTextOption{html.String(nil, text)}
	}
	return []message.StyledTextOption{styling.Plain(text)}
}

// SendMessage sends text to a chat and returns the new message ID.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, opts ...SendOption) (int, error) {
	o, err := c.prepare(ctx, chatID, opts)
	if err != nil {
		return 0, err
	}
	return unpack.MessageID(o.b.StyledText(ctx, o.styled(text)...))
}

// Send posts text into the chat the update came from.
func (x *Context) Send(text string, opts ...SendOption) (int, error) {
	return x.Client.SendMessage(x, x.chatID, text, opts...)
}
