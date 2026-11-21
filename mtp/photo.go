package mtp

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/unpack"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

// Photo is an image to send. Build one with PhotoFile, PhotoURL or
// PhotoBytes.
type Photo struct {
	path string
	url  string
	name string
	data []byte
}

// PhotoFile sends an image from the local filesystem.
func PhotoFile(path string) Photo { return Photo{path: path} }

// PhotoURL sends an image from a public http(s) URL.
func PhotoURL(url string) Photo { return Photo{url: url} }

// PhotoBytes sends an image held in memory; name is the file name
// Telegram sees, e.g. "chart.png".
func PhotoBytes(name string, data []byte) Photo { return Photo{name: name, data: data} }

// upload pushes the image bytes to Telegram and returns the file handle.
func (c *Client) upload(ctx context.Context, p Photo) (tg.InputFileClass, error) {
	up := uploader.NewUploader(c.api)
	switch {
	case p.path != "":
		return up.FromPath(ctx, p.path)
	case p.url != "":
		return up.FromURL(ctx, p.url)
	case len(p.data) > 0:
		return up.FromBytes(ctx, p.name, p.data)
	}
	return nil, errors.New("mtp: empty Photo")
}

// SendPhoto sends an image with an optional caption and returns the new
// message ID. All SendOptions apply; HTML() formats the caption.
func (c *Client) SendPhoto(ctx context.Context, chatID int64, p Photo, caption string, opts ...SendOption) (int, error) {
	o, err := c.prepare(ctx, chatID, opts)
	if err != nil {
		return 0, err
	}
	var cap []message.StyledTextOption
	if caption != "" {
		cap = o.styled(caption)
	}

	// For URLs, first let Telegram fetch the image itself: nothing has to
	// pass through this process. Fall back to download-and-upload when
	// Telegram cannot reach or parse the URL.
	if p.url != "" {
		id, err := unpack.MessageID(o.b.PhotoExternal(ctx, p.url, cap...))
		if err == nil {
			return id, nil
		}
		c.log.Debug("photo by URL failed, uploading instead", "err", err)
	}

	file, err := c.upload(ctx, p)
	if err != nil {
		return 0, fmt.Errorf("mtp: upload photo: %w", err)
	}
	return unpack.MessageID(o.b.UploadedPhoto(ctx, file, cap...))
}

// SendPhoto posts an image into the chat the update came from.
func (x *Context) SendPhoto(p Photo, caption string, opts ...SendOption) (int, error) {
	return x.Client.SendPhoto(x, x.chatID, p, caption, opts...)
}
