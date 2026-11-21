package mtp

// ReplyTo makes the outgoing message a reply to messageID in the same chat.
func ReplyTo(messageID int) SendOption {
	return func(o *sendOptions) {
		if messageID != 0 {
			o.b = o.b.Reply(messageID)
		}
	}
}

// replyOpts puts ReplyTo first so a caller-supplied ReplyTo still wins.
func (x *Context) replyOpts(opts []SendOption) []SendOption {
	return append([]SendOption{ReplyTo(x.msgID)}, opts...)
}

// Reply answers the incoming message as a quoted reply. In an OnCallback
// handler it replies to the message that carries the tapped button.
func (x *Context) Reply(text string, opts ...SendOption) (int, error) {
	return x.Client.SendMessage(x, x.chatID, text, x.replyOpts(opts)...)
}

// ReplyPhoto answers the incoming message with an image as a quoted reply.
func (x *Context) ReplyPhoto(p Photo, caption string, opts ...SendOption) (int, error) {
	return x.Client.SendPhoto(x, x.chatID, p, caption, x.replyOpts(opts)...)
}
