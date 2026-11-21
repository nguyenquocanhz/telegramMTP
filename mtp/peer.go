package mtp

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/td/tg"
)

// channelOffset turns a channel ID into its -100<id> form.
const channelOffset = 1_000_000_000_000

type peerKind int

const (
	kindUser peerKind = iota
	kindChat
	kindChannel
)

// chatIDOf converts a raw peer into the single-int64 chat ID convention.
func chatIDOf(p tg.PeerClass) int64 {
	switch v := p.(type) {
	case *tg.PeerUser:
		return v.UserID
	case *tg.PeerChat:
		return -v.ChatID
	case *tg.PeerChannel:
		return -(channelOffset + v.ChannelID)
	}
	return 0
}

// splitChatID is the inverse of chatIDOf.
func splitChatID(chatID int64) (peerKind, int64) {
	switch {
	case chatID > 0:
		return kindUser, chatID
	case chatID < -channelOffset:
		return kindChannel, -chatID - channelOffset
	default:
		return kindChat, -chatID
	}
}

// peerCache remembers access hashes seen in updates. MTProto needs them
// to address users and channels; without the cache every send would cost
// an extra lookup round trip.
type peerCache struct {
	mu sync.RWMutex
	m  map[int64]tg.InputPeerClass
}

func newPeerCache() *peerCache {
	return &peerCache{m: make(map[int64]tg.InputPeerClass)}
}

func (p *peerCache) get(chatID int64) (tg.InputPeerClass, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	peer, ok := p.m[chatID]
	return peer, ok
}

func (p *peerCache) put(chatID int64, peer tg.InputPeerClass) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m[chatID] = peer
}

func (p *peerCache) learn(e tg.Entities) {
	if len(e.Users) == 0 && len(e.Channels) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, u := range e.Users {
		// "min" users carry an access hash that only works in the context
		// they were seen in, so they are not safe to cache.
		if !u.Min {
			p.m[id] = u.AsInputPeer()
		}
	}
	for id, ch := range e.Channels {
		if !ch.Min {
			p.m[-(channelOffset + id)] = ch.AsInputPeer()
		}
	}
}

// resolve turns a chat ID into an input peer, asking Telegram for the
// access hash when it has not been seen yet (bots may do that with hash 0).
func (c *Client) resolve(ctx context.Context, chatID int64) (tg.InputPeerClass, error) {
	if chatID == 0 {
		return nil, fmt.Errorf("mtp: chat ID is 0")
	}
	if peer, ok := c.peers.get(chatID); ok {
		return peer, nil
	}

	kind, id := splitChatID(chatID)
	switch kind {
	case kindChat:
		return &tg.InputPeerChat{ChatID: id}, nil

	case kindUser:
		users, err := c.api.UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUser{UserID: id}})
		if err != nil {
			return nil, fmt.Errorf("mtp: resolve user %d: %w", id, err)
		}
		for _, uc := range users {
			if u, ok := uc.(*tg.User); ok && u.ID == id {
				peer := u.AsInputPeer()
				c.peers.put(chatID, peer)
				return peer, nil
			}
		}
		return nil, fmt.Errorf("mtp: user %d not found (they must message the bot first)", id)

	default:
		res, err := c.api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: id}})
		if err != nil {
			return nil, fmt.Errorf("mtp: resolve channel %d: %w", id, err)
		}
		for _, cc := range res.GetChats() {
			if ch, ok := cc.(*tg.Channel); ok && ch.ID == id {
				peer := ch.AsInputPeer()
				c.peers.put(chatID, peer)
				return peer, nil
			}
		}
		return nil, fmt.Errorf("mtp: channel %d not found (the bot must be a member)", id)
	}
}
