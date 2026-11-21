package mtp

import (
	"context"
	"strings"
	"time"

	"github.com/gotd/td/tg"
)

// Message is an incoming message reduced to the fields handlers usually need.
type Message struct {
	ID       int
	ChatID   int64
	SenderID int64
	Text     string
	// ReplyToID is the ID of the message this one replies to, or 0.
	ReplyToID int
	Date      time.Time
	// Raw is the full MTProto message for anything not surfaced above.
	Raw *tg.Message
}

func newMessage(m *tg.Message) *Message {
	msg := &Message{
		ID:     m.ID,
		ChatID: chatIDOf(m.PeerID),
		Text:   m.Message,
		Date:   time.Unix(int64(m.Date), 0),
		Raw:    m,
	}
	// In private chats Telegram omits FromID: the sender is the peer itself.
	if m.FromID != nil {
		msg.SenderID = chatIDOf(m.FromID)
	} else {
		msg.SenderID = msg.ChatID
	}
	if h, ok := m.ReplyTo.(*tg.MessageReplyHeader); ok {
		msg.ReplyToID = h.ReplyToMsgID
	}
	return msg
}

// Context is passed to every handler. It embeds the request context, so
// it can be handed straight to anything that wants a context.Context.
type Context struct {
	context.Context
	Client *Client
	// Message is the incoming message.
	Message *Message
	// Command is the matched command name without the slash ("" for
	// plain text) and Args is the rest of the line.
	Command string
	Args    string

	chatID   int64
	senderID int64
	msgID    int
}

// ChatID is the chat the update happened in.
func (x *Context) ChatID() int64 { return x.chatID }

// SenderID is the user who triggered the update.
func (x *Context) SenderID() int64 { return x.senderID }

// Text is the incoming message text ("" when there is no message).
func (x *Context) Text() string {
	if x.Message == nil {
		return ""
	}
	return x.Message.Text
}

// ArgList splits Args on whitespace.
func (x *Context) ArgList() []string { return strings.Fields(x.Args) }
