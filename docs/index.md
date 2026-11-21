---
title: Giới thiệu
layout: default
nav_order: 1
permalink: /
---

# telegramMTP
{: .fs-9 }

Thư viện Go nói chuyện với Telegram trực tiếp qua MTProto, với cách dùng gọn như Telethon.
{: .fs-6 .fw-300 }

[Bắt đầu nhanh](bat-dau.md){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[Xem trên GitHub](https://github.com/nguyenquocanhz/telegramMTP){: .btn .fs-5 .mb-4 .mb-md-0 }

---

`telegramMTP` không đi qua Bot API HTTP. Nó kết nối thẳng tới Telegram bằng MTProto nhờ [gotd/td](https://github.com/gotd/td), rồi giấu các kiểu TL thô sau vài hàm dễ nhớ:

```go
bot, err := mtp.New(mtp.Config{APIID: id, APIHash: hash, BotToken: token})
if err != nil {
    log.Fatal(err)
}

bot.OnCommand("ping", "Kiểm tra bot", func(c *mtp.Context) error {
    _, err := c.Reply("pong")
    return err
})

log.Fatal(bot.Run(ctx))
```

Module gồm hai package dùng độc lập được, và một bot mẫu:

| Thành phần | Đường dẫn import | Việc làm |
| --- | --- | --- |
| `mtp` | `github.com/nguyenquocanhz/telegramMTP/mtp` | Client, bắt lệnh, gửi tin, bàn phím, ảnh, hàng đợi, admin |
| `secret` | `github.com/nguyenquocanhz/telegramMTP/secret` | Nạp bí mật từ nhiều nguồn, kho mã hoá |
| `natterbot` | `cmd/natterbot` | Bot chạy được ngay, mỗi lệnh minh hoạ một tính năng |

## Tính năng

| Tính năng | Gọi thế nào | Trang |
| --- | --- | --- |
| Gửi và trả lời tin | `c.Send(text)` · `c.Reply(text)` · `bot.SendMessage(ctx, chatID, text)` | [Gửi tin](gui-tin.md) |
| Bắt lệnh | `bot.OnCommand("start", "mô tả", h)` | [Lệnh và handler](lenh.md) |
| Bàn phím lệnh | `mtp.WithKeyboard(bot.CommandKeyboard(3))` | [Bàn phím lệnh](ban-phim.md) |
| Nút inline | `mtp.WithInline(...)` + `bot.OnCallback(prefix, h)` | [Nút inline](nut-inline.md) |
| Gửi ảnh | `c.SendPhoto(mtp.PhotoFile(path), caption)` | [Gửi ảnh](anh.md) |
| Hàng đợi | `bot.QueueMessage(chatID, text)` | [Hàng đợi](hang-doi.md) |
| Admin và danh sách lệnh | `Config.AdminIDs` + `bot.OnAdminCommand(...)` | [Admin](admin.md) |
| Bí mật | package `secret` | [SecretManager](secret.md) |

## Nếu bạn đã quen Telethon

| Telethon | telegramMTP |
| --- | --- |
| `client.send_message(chat, text)` | `bot.SendMessage(ctx, chatID, text)` |
| `event.respond(text)` | `c.Send(text)` |
| `event.reply(text)` | `c.Reply(text)` |
| `client.send_file(chat, file)` | `bot.SendPhoto(ctx, chatID, mtp.PhotoFile(path), caption)` |
| `@client.on(events.NewMessage(pattern="/start"))` | `bot.OnCommand("start", "mô tả", h)` |
| `Button.text(...)` | `mtp.NewKeyboard(mtp.KeyRow(...))` |
| `Button.inline(text, data)` + `events.CallbackQuery` | `mtp.Btn(text, data)` + `bot.OnCallback(prefix, h)` |
| `event.answer()` · `event.edit()` | `c.Answer(text)` · `c.Edit(text)` |
| `client.run_until_disconnected()` | `bot.Run(ctx)` |

Chat được định danh bằng một số `int64` theo đúng quy ước "marked ID" của Telethon: user là số dương, nhóm thường là số âm, kênh và supergroup là `-100<id>`. Thư viện tự lo access hash.

## Giới hạn hiện tại

- Chỉ đăng nhập bằng **bot token**. Đăng nhập tài khoản người dùng (số điện thoại + mã) chưa được bọc.
- Định dạng hỗ trợ văn bản thường và HTML (`mtp.HTML()`), chưa có Markdown.
- Tệp gửi đi mới có ảnh; tài liệu, video, album chưa được bọc.
- Hàng đợi nằm trong bộ nhớ: tin đang chờ sẽ mất khi tiến trình dừng.

Việc gì chưa được bọc vẫn gọi thẳng được qua `bot.API()`, trả về client gotd thô.
