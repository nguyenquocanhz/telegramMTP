---
title: Nút inline
layout: default
nav_order: 6
---

# Nút inline

Nút inline nằm ngay dưới một tin nhắn. Bấm nút không gửi gì vào chat; thay vào đó bot nhận một *callback* mang chuỗi dữ liệu của nút.

## Gắn nút

```go
bot.OnCommand("vote", "Bình chọn", func(c *mtp.Context) error {
	_, err := c.Send("Bạn chọn gì?", mtp.WithInline(
		mtp.InlineRow(mtp.Btn("👍 Có", "vote:yes"), mtp.Btn("👎 Không", "vote:no")),
		mtp.InlineRow(mtp.URLBtn("Xem chi tiết", "https://example.com")),
	))
	return err
})
```

| Hàm | Tác dụng |
| --- | --- |
| `mtp.Btn(text, data)` | Nút callback: bấm vào thì `data` được chuyển tới handler |
| `mtp.URLBtn(text, url)` | Nút mở liên kết, không sinh callback |
| `mtp.InlineRow(buttons...)` | Một hàng nút |

{: .warning }
Telegram giới hạn `data` ở 64 byte. Chỉ nên để một mã ngắn trong đó (ví dụ `order:1024`), dữ liệu thật giữ ở phía bot.

## Bắt lần bấm

`OnCallback` khớp theo **tiền tố** của `data`:

```go
bot.OnCallback("vote:", func(c *mtp.Context) error {
	choice := strings.TrimPrefix(c.Callback.Data, "vote:")
	if err := c.Answer("Đã ghi nhận: " + choice); err != nil {
		return err
	}
	return c.Edit("Cảm ơn bạn đã bình chọn.")
})
```

Khi nhiều tiền tố cùng khớp, tiền tố dài nhất thắng. Tiền tố rỗng `""` bắt mọi callback còn lại, tiện làm nhánh mặc định.

Trong handler của callback, `c.Message` là `nil` và `c.Callback` chứa:

| Trường | Ý nghĩa |
| --- | --- |
| `Data` | Chuỗi đã truyền cho `Btn` |
| `MessageID` | Tin mang nút vừa bấm |
| `QueryID` | ID của lần bấm |

`c.ChatID()` và `c.SenderID()` vẫn dùng được như thường.

## Phản hồi lần bấm

| Method | Người dùng thấy |
| --- | --- |
| `c.Answer(text)` | Dòng thông báo ngắn tự biến mất. `text` rỗng thì chỉ tắt vòng xoay chờ |
| `c.AnswerAlert(text)` | Hộp thoại phải bấm OK |
| `c.Edit(text, opts...)` | Thay nội dung tin mang nút |
| `c.Reply(text)` | Gửi tin mới, trích dẫn tin mang nút |
| `c.Send(text)` | Gửi tin mới vào chat |

Nếu handler không gọi `Answer` hay `AnswerAlert`, thư viện tự xác nhận lần bấm sau khi handler chạy xong, nên nút không bao giờ bị kẹt ở trạng thái chờ.

## Menu nhiều tầng

`Edit` xoá các nút cũ trừ khi bạn truyền lại `WithInline`. Tận dụng điều đó để chuyển màn hình ngay trong một tin:

```go
func mainMenu() mtp.SendOption {
	return mtp.WithInline(
		mtp.InlineRow(mtp.Btn("ℹ️ Giới thiệu", "menu:about")),
		mtp.InlineRow(mtp.Btn("✖ Đóng", "menu:close")),
	)
}

bot.OnCommand("menu", "Mở menu", func(c *mtp.Context) error {
	_, err := c.Send("📋 Menu chính", mainMenu())
	return err
})
bot.OnCallback("menu:about", func(c *mtp.Context) error {
	return c.Edit("Bot viết bằng telegramMTP.",
		mtp.WithInline(mtp.InlineRow(mtp.Btn("⬅ Quay lại", "menu:home"))))
})
bot.OnCallback("menu:home", func(c *mtp.Context) error {
	return c.Edit("📋 Menu chính", mainMenu())
})
bot.OnCallback("menu:close", func(c *mtp.Context) error {
	return c.Edit("Menu đã đóng.") // không truyền WithInline: nút biến mất
})
```

Ngoài handler, dùng `bot.EditMessage(ctx, chatID, messageID, text, opts...)` để sửa một tin bất kỳ của bot.
