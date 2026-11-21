package main

import (
	"fmt"
	"html"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/nguyenquocanhz/telegramMTP/mtp"
)

var startedAt = time.Now()

// registerHandlers wires up the bot's command list. The order here is
// the order shown in Telegram's "/" menu and on the command keyboard.
func registerHandlers(bot *mtp.Client) {
	// --- sendMessage + CommandKeyboard -------------------------------
	bot.OnCommand("start", "Bắt đầu và hiện bàn phím lệnh", func(c *mtp.Context) error {
		_, err := c.Send(
			"👋 Xin chào! Tôi là bot chạy trực tiếp trên MTProto.\nChọn một lệnh bên dưới hoặc gõ /help.",
			mtp.WithKeyboard(bot.CommandKeyboard(3).Placeholder("Chọn lệnh…")),
		)
		return err
	})

	bot.OnCommand("help", "Danh sách lệnh", func(c *mtp.Context) error {
		var b strings.Builder
		b.WriteString("<b>Danh sách lệnh</b>\n")
		for _, cmd := range bot.Commands() {
			if cmd.Description == "" || (cmd.AdminOnly && !bot.IsAdmin(c.SenderID())) {
				continue
			}
			fmt.Fprintf(&b, "/%s — %s", cmd.Name, html.EscapeString(cmd.Description))
			if cmd.AdminOnly {
				b.WriteString(" <i>(admin)</i>")
			}
			b.WriteString("\n")
		}
		_, err := c.Send(b.String(), mtp.HTML())
		return err
	})

	bot.OnCommand("id", "Xem ID của bạn (dùng cho ADMIN_ID)", func(c *mtp.Context) error {
		text := fmt.Sprintf("👤 User ID: <code>%d</code>\n💬 Chat ID: <code>%d</code>", c.SenderID(), c.ChatID())
		if bot.IsAdmin(c.SenderID()) {
			text += "\n⭐ Bạn là admin."
		}
		_, err := c.Reply(text, mtp.HTML())
		return err
	})

	// --- replyMessage ------------------------------------------------
	bot.OnCommand("ping", "Kiểm tra bot còn sống", func(c *mtp.Context) error {
		began := time.Now()
		if _, err := c.Reply("🏓 pong"); err != nil {
			return err
		}
		_, err := c.Send(fmt.Sprintf("Độ trễ gửi: %d ms", time.Since(began).Milliseconds()), mtp.Silent())
		return err
	})

	bot.OnCommand("echo", "Nhắc lại nội dung: /echo xin chào", func(c *mtp.Context) error {
		if c.Args == "" {
			_, err := c.Reply("Cách dùng: /echo <nội dung>")
			return err
		}
		_, err := c.Reply(c.Args)
		return err
	})

	// --- InlineKeyboard ----------------------------------------------
	bot.OnCommand("menu", "Mở menu nút bấm", func(c *mtp.Context) error {
		_, err := c.Send("📋 Menu chính", mainMenu())
		return err
	})
	bot.OnCallback("menu:photo", func(c *mtp.Context) error {
		if err := c.Answer("Đang gửi ảnh…"); err != nil {
			return err
		}
		_, err := c.SendPhoto(demoPhoto(), "🖼 Ảnh gửi từ nút bấm")
		return err
	})
	bot.OnCallback("menu:about", func(c *mtp.Context) error {
		return c.Edit(
			"<b>telegramMTP</b> — lớp bọc MTProto viết bằng Go, lấy cảm hứng từ Telethon.",
			mtp.HTML(),
			mtp.WithInline(mtp.InlineRow(mtp.Btn("⬅ Quay lại", "menu:home"))),
		)
	})
	bot.OnCallback("menu:home", func(c *mtp.Context) error {
		return c.Edit("📋 Menu chính", mainMenu())
	})
	bot.OnCallback("menu:close", func(c *mtp.Context) error {
		if err := c.AnswerAlert("Đã đóng menu"); err != nil {
			return err
		}
		return c.Edit("Menu đã đóng. Gõ /menu để mở lại.")
	})

	// --- addImage ----------------------------------------------------
	bot.OnCommand("photo", "Gửi một ảnh mẫu", func(c *mtp.Context) error {
		_, err := c.ReplyPhoto(demoPhoto(), "🖼 Ảnh được tạo trong bộ nhớ rồi tải lên qua MTProto")
		return err
	})

	// --- queueMessage ------------------------------------------------
	bot.OnCommand("queue", "Thử hàng đợi: /queue 5", func(c *mtp.Context) error {
		n := 5
		if c.Args != "" {
			v, err := strconv.Atoi(c.ArgList()[0])
			if err != nil || v < 1 || v > 20 {
				_, err := c.Reply("Số tin phải từ 1 đến 20.")
				return err
			}
			n = v
		}
		chat := c.ChatID()
		for i := 1; i <= n; i++ {
			bot.QueueMessage(chat, fmt.Sprintf("📨 Tin %d/%d từ hàng đợi", i, n), mtp.Silent())
		}
		_, err := c.Reply(fmt.Sprintf("Đã xếp %d tin vào hàng đợi, mỗi giây gửi một tin.", n))
		return err
	})

	bot.OnCommand("hide", "Ẩn bàn phím lệnh", func(c *mtp.Context) error {
		_, err := c.Send("Đã ẩn bàn phím. Gõ /start để hiện lại.", mtp.RemoveKeyboard())
		return err
	})

	// --- ADMIN_ID ----------------------------------------------------
	bot.OnAdminCommand("stats", "Trạng thái bot", func(c *mtp.Context) error {
		text := fmt.Sprintf(
			"<b>Trạng thái</b>\nPhiên bản: <code>%s</code>\nUptime: %s\nHàng đợi: %d tin\nGoroutine: %d\nGo: %s",
			html.EscapeString(version),
			time.Since(startedAt).Round(time.Second),
			bot.Queue().Len(),
			runtime.NumGoroutine(),
			runtime.Version(),
		)
		_, err := c.Reply(text, mtp.HTML())
		return err
	})

	bot.OnAdminCommand("say", "Gửi tin tới chat: /say <chat_id> <nội dung>", func(c *mtp.Context) error {
		target, text, _ := strings.Cut(c.Args, " ")
		chatID, err := strconv.ParseInt(target, 10, 64)
		if text = strings.TrimSpace(text); err != nil || text == "" {
			_, err := c.Reply("Cách dùng: /say <chat_id> <nội dung>")
			return err
		}
		if err := <-bot.QueueMessage(chatID, text); err != nil {
			_, err := c.Reply("❌ Không gửi được: " + err.Error())
			return err
		}
		_, err = c.Reply("✅ Đã gửi.")
		return err
	})

	// Plain text in a private chat gets a hint; groups are left alone.
	bot.OnMessage(func(c *mtp.Context) error {
		if c.ChatID() <= 0 || c.Text() == "" {
			return nil
		}
		_, err := c.Reply("Tôi chưa hiểu. Gõ /help để xem các lệnh.")
		return err
	})
}

func mainMenu() mtp.SendOption {
	return mtp.WithInline(
		mtp.InlineRow(mtp.Btn("🖼 Gửi ảnh", "menu:photo"), mtp.Btn("ℹ️ Giới thiệu", "menu:about")),
		mtp.InlineRow(mtp.URLBtn("🌐 Telegram MTProto", "https://core.telegram.org/mtproto")),
		mtp.InlineRow(mtp.Btn("✖ Đóng", "menu:close")),
	)
}
