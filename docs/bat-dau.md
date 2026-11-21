---
title: Bắt đầu nhanh
layout: default
nav_order: 2
---

# Bắt đầu nhanh

Từ con số không tới một bot trả lời `/ping`, trong khoảng năm phút.

## 1. Chuẩn bị

Bạn cần Go 1.25 trở lên và ba giá trị:

| Giá trị | Lấy ở đâu |
| --- | --- |
| `API_ID`, `API_HASH` | <https://my.telegram.org/apps>, tạo một "application" |
| `BOT_TOKEN` | Nhắn `/newbot` cho [@BotFather](https://t.me/BotFather) |

Khác với Bot API, MTProto cần cả `API_ID` và `API_HASH` ngoài bot token.

## 2. Cài thư viện

```powershell
go mod init example.com/mybot
```

```powershell
go get github.com/nguyenquocanhz/telegramMTP@latest
```

## 3. Viết bot

Tạo `main.go`:

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/nguyenquocanhz/telegramMTP/mtp"
	"github.com/nguyenquocanhz/telegramMTP/secret"
)

func main() {
	// Đọc cấu hình từ biến môi trường hoặc file .env.
	secrets, err := secret.New(secret.Options{})
	if err != nil {
		log.Fatal(err)
	}
	apiID, err := secrets.Int64("API_ID")
	if err != nil {
		log.Fatal(err)
	}
	apiHash, err := secrets.Require("API_HASH")
	if err != nil {
		log.Fatal(err)
	}
	token, err := secrets.Require("BOT_TOKEN")
	if err != nil {
		log.Fatal(err)
	}

	bot, err := mtp.New(mtp.Config{
		APIID:    int(apiID),
		APIHash:  apiHash,
		BotToken: token,
	})
	if err != nil {
		log.Fatal(err)
	}

	bot.OnCommand("start", "Bắt đầu", func(c *mtp.Context) error {
		_, err := c.Send("Xin chào! Gõ /ping để thử.")
		return err
	})
	bot.OnCommand("ping", "Kiểm tra bot", func(c *mtp.Context) error {
		_, err := c.Reply("pong")
		return err
	})

	// Ctrl+C dừng bot gọn gàng.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := bot.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

## 4. Điền cấu hình và chạy

Tạo file `.env` cạnh `main.go`:

```
API_ID=1234567
API_HASH=0123456789abcdef0123456789abcdef
BOT_TOKEN=123456789:AA...
```

```powershell
go run .
```

Mở Telegram, nhắn `/start` cho bot. Gõ `/` sẽ thấy hai lệnh vừa đăng ký, vì danh sách lệnh được đẩy lên Telegram mỗi lần bot khởi động.

{: .warning }
`.env` và `session.json` chứa thông tin đăng nhập. Thêm cả hai vào `.gitignore` trước khi commit.

## Lần chạy đầu tiên làm gì

1. `mtp.New` kiểm tra cấu hình và đăng ký handler. Chưa có kết nối mạng nào.
2. `bot.Run` kết nối MTProto, đăng nhập bằng bot token rồi lưu khoá phiên vào `session.json` (đổi chỗ bằng `Config.SessionFile`).
3. Những lần chạy sau dùng lại `session.json`, không đăng nhập lại.
4. `Run` chặn cho tới khi `ctx` bị huỷ, và trả về `nil` khi dừng bình thường.

Chi tiết từng bước có ở trang [Sơ đồ luồng](luong.md).

## Chạy bot mẫu natterbot

Muốn xem mọi tính năng chạy thật trước khi viết bot của mình thì dùng `natterbot` trong repo:

```powershell
git clone https://github.com/nguyenquocanhz/telegramMTP.git
```

```powershell
cd telegramMTP; Copy-Item .env.example .env
```

Điền `API_ID`, `API_HASH`, `BOT_TOKEN` vào `.env`, rồi:

```powershell
go run ./cmd/natterbot
```

| Lệnh | Việc làm | Minh hoạ |
| --- | --- | --- |
| `/start` | Chào và hiện bàn phím lệnh | [Gửi tin](gui-tin.md), [Bàn phím lệnh](ban-phim.md) |
| `/help` | Liệt kê lệnh | [Admin và danh sách lệnh](admin.md) |
| `/id` | Xem User ID và Chat ID | [Gửi tin](gui-tin.md) |
| `/ping`, `/echo <nội dung>` | Trả lời tin | [Gửi tin](gui-tin.md) |
| `/menu` | Menu nút bấm | [Nút inline](nut-inline.md) |
| `/photo` | Gửi ảnh mẫu | [Gửi ảnh](anh.md) |
| `/queue [n]` | Xếp n tin vào hàng đợi | [Hàng đợi](hang-doi.md) |
| `/hide` | Ẩn bàn phím lệnh | [Bàn phím lệnh](ban-phim.md) |
| `/stats`, `/say <chat_id> <nội dung>` | Lệnh chỉ admin chạy được | [Admin và danh sách lệnh](admin.md) |

Mã của các lệnh này nằm trong [`cmd/natterbot/handlers.go`](https://github.com/nguyenquocanhz/telegramMTP/blob/main/cmd/natterbot/handlers.go).
