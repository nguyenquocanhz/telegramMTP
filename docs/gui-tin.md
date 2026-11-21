---
title: Gửi tin
layout: default
nav_order: 3
---

# Gửi và trả lời tin

## Trong handler

Trong một handler, `Context` đã biết update đến từ chat nào:

```go
bot.OnCommand("hello", "Chào", func(c *mtp.Context) error {
	// Gửi một tin vào chat hiện tại.
	if _, err := c.Send("Xin chào!"); err != nil {
		return err
	}
	// Trả lời kèm trích dẫn tin của người dùng.
	_, err := c.Reply("Đây là câu trả lời cho tin của bạn.")
	return err
})
```

Cả hai trả về ID của tin vừa gửi, dùng được sau này cho `ReplyTo` hoặc `EditMessage`.

## Ngoài handler

Khi không có `Context` (ví dụ trong một tác vụ hẹn giờ), gọi thẳng trên client với chat ID:

```go
id, err := bot.SendMessage(ctx, chatID, "Báo cáo hằng ngày đã sẵn sàng")
```

Để gửi nhiều tin liên tiếp, dùng [hàng đợi](hang-doi.md) thay vì gọi `SendMessage` trong vòng lặp.

{: .note }
Bot chỉ nhắn được cho người đã từng mở chat với nó (đã bấm Start). Đây là quy định của Telegram, không phải của thư viện.

## Chat ID

Mọi hàm nhận một `int64` duy nhất, cùng quy ước với Telethon và Bot API:

| Loại chat | Chat ID | Ví dụ |
| --- | --- | --- |
| Người dùng | số dương | `123456789` |
| Nhóm thường | số âm | `-987654321` |
| Kênh, supergroup | `-100` ghép với ID | `-1001234567890` |

Access hash mà MTProto đòi hỏi được thư viện ghi nhớ từ các update đi qua, nên bạn không phải quan tâm tới nó.

## Tuỳ chọn gửi

Mọi hàm gửi nhận thêm các `SendOption` ở cuối, kết hợp tuỳ ý:

```go
c.Send("<b>Đậm</b> và <i>nghiêng</i>", mtp.HTML(), mtp.Silent())
```

| Tuỳ chọn | Tác dụng |
| --- | --- |
| `mtp.HTML()` | Hiểu văn bản là HTML của Telegram: `<b>`, `<i>`, `<code>`, `<a href>`… |
| `mtp.Silent()` | Gửi không kèm âm báo |
| `mtp.NoPreview()` | Tắt xem trước liên kết |
| `mtp.ReplyTo(messageID)` | Biến tin thành câu trả lời cho `messageID` trong cùng chat |
| `mtp.WithKeyboard(k)` | Gắn [bàn phím lệnh](ban-phim.md) |
| `mtp.RemoveKeyboard()` | Ẩn bàn phím lệnh đã hiện trước đó |
| `mtp.WithInline(rows...)` | Gắn [nút inline](nut-inline.md) |

{: .warning }
Khi dùng `mtp.HTML()` với nội dung do người dùng nhập, hãy bọc nó bằng `html.EscapeString` để ký tự `<` và `&` không phá định dạng.

## Trả lời một tin bất kỳ

`c.Reply` luôn trả lời tin đang xử lý. Để trả lời tin khác, dùng `ReplyTo` với ID của nó:

```go
first, err := c.Send("Tin gốc")
if err != nil {
	return err
}
_, err = c.Send("Trả lời tin gốc", mtp.ReplyTo(first))
```

Tin đến có trả lời tin nào thì `c.Message.ReplyToID` chứa ID tin đó, ngược lại là `0`.

## Sửa tin đã gửi

```go
id, err := c.Send("Đang xử lý…")
if err != nil {
	return err
}
// ... làm việc ...
err = bot.EditMessage(c, c.ChatID(), id, "Xong ✅")
```

`Context` chính là một `context.Context`, nên truyền thẳng `c` vào chỗ cần `ctx`. Bot chỉ sửa được tin do chính nó gửi.
