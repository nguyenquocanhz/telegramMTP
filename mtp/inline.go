package mtp

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
)

// InlineButton is a button attached under a message.
type InlineButton struct {
	text string
	data string
	url  string
}

// Btn is a callback button: tapping it delivers data to the matching
// OnCallback handler. Telegram limits data to 64 bytes.
func Btn(text, data string) InlineButton { return InlineButton{text: text, data: data} }

// URLBtn opens a link when tapped.
func URLBtn(text, url string) InlineButton { return InlineButton{text: text, url: url} }

// InlineRow is one row of inline buttons.
func InlineRow(buttons ...InlineButton) []InlineButton { return buttons }

func inlineMarkup(rows [][]InlineButton) tg.ReplyMarkupClass {
	out := make([]tg.KeyboardInlineButtonRow, 0, len(rows))
	for _, row := range rows {
		buttons := make([]tg.KeyboardInlineButton, 0, len(row))
		for _, b := range row {
			if b.url != "" {
				buttons = append(buttons, markup.URL(b.text, b.url))
			} else {
				buttons = append(buttons, markup.Callback(b.text, []byte(b.data)))
			}
		}
		out = append(out, markup.InlineButtonRow(buttons...))
	}
	return markup.InlineKeyboard(out...)
}

// WithInline attaches an inline keyboard to the message.
//
//	c.Send("Pick one", mtp.WithInline(
//		mtp.InlineRow(mtp.Btn("Yes", "vote:yes"), mtp.Btn("No", "vote:no")),
//	))
func WithInline(rows ...[]InlineButton) SendOption {
	return func(o *sendOptions) { o.b = o.b.Markup(inlineMarkup(rows)) }
}

// Callback describes a tap on an inline button.
type Callback struct {
	QueryID int64
	// Data is the string given to Btn.
	Data string
	// MessageID is the message the button is attached to.
	MessageID int

	answered atomic.Bool
}

type callbackRoute struct {
	prefix string
	h      Handler
}

// OnCallback registers a handler for inline buttons whose data starts
// with prefix. The longest matching prefix wins; "" catches everything.
func (c *Client) OnCallback(prefix string, h Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callbacks = append(c.callbacks, callbackRoute{prefix: prefix, h: h})
}

func (c *Client) callbackHandler(data string) Handler {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var best Handler
	bestLen := -1
	for _, r := range c.callbacks {
		if strings.HasPrefix(data, r.prefix) && len(r.prefix) > bestLen {
			best, bestLen = r.h, len(r.prefix)
		}
	}
	return best
}

func (c *Client) handleCallback(ctx context.Context, e tg.Entities, u *tg.UpdateBotCallbackQuery) {
	c.peers.learn(e)
	x := &Context{
		Context:  ctx,
		Client:   c,
		Callback: &Callback{QueryID: u.QueryID, Data: string(u.Data), MessageID: u.MsgID},
		chatID:   chatIDOf(u.Peer),
		senderID: u.UserID,
		msgID:    u.MsgID,
	}
	go func() {
		if h := c.callbackHandler(x.Callback.Data); h != nil {
			c.runHandlers(x, []Handler{h})
		}
		// Telegram shows a loading spinner on the button until the query
		// is answered, so answer for handlers that did not.
		if !x.Callback.answered.Load() {
			if err := x.Answer(""); err != nil {
				c.log.Warn("answer callback", "err", err)
			}
		}
	}()
}

// Answer acknowledges a button tap, optionally showing text as a brief
// toast. Unanswered taps are acknowledged automatically after the handler.
func (x *Context) Answer(text string) error { return x.answer(text, false) }

// AnswerAlert acknowledges a button tap with a modal popup.
func (x *Context) AnswerAlert(text string) error { return x.answer(text, true) }

func (x *Context) answer(text string, alert bool) error {
	if x.Callback == nil {
		return errors.New("mtp: Answer is only valid inside an OnCallback handler")
	}
	x.Callback.answered.Store(true)
	_, err := x.Client.api.MessagesSetBotCallbackAnswer(x, &tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: x.Callback.QueryID,
		Message: text,
		Alert:   alert,
	})
	return err
}

// EditMessage replaces the text of a message the bot sent. Pass WithInline
// again if the edited message should keep its buttons.
func (c *Client) EditMessage(ctx context.Context, chatID int64, messageID int, text string, opts ...SendOption) error {
	o, err := c.prepare(ctx, chatID, opts)
	if err != nil {
		return err
	}
	_, err = o.b.Edit(messageID).StyledText(ctx, o.styled(text)...)
	return err
}

// Edit replaces the text of the message whose button was tapped.
func (x *Context) Edit(text string, opts ...SendOption) error {
	if x.Callback == nil {
		return errors.New("mtp: Edit is only valid inside an OnCallback handler")
	}
	return x.Client.EditMessage(x, x.chatID, x.Callback.MessageID, text, opts...)
}
