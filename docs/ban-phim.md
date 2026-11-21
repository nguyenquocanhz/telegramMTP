---
title: Bàn phím lệnh
layout: default
nav_order: 5
---

# Bàn phím lệnh

Bàn phím lệnh (reply keyboard) là hàng nút hiện thay cho bàn phím chữ của người dùng. Bấm một nút sẽ **gửi nhãn của nút như một tin nhắn**, nên nút mang nhãn `/help` chạy đúng lệnh `/help`.

## Tự dựng từ các lệnh đã đăng ký

```go
bot.OnCommand("start", "Bắt đầu", func(c *mtp.Context) error {
	_, err := c.Send("Chọn một lệnh:",
		mtp.WithKeyboard(bot.CommandKeyboard(3)))
	return err
})
```

`bot.CommandKeyboard(3)` xếp các lệnh đăng ký bằng `OnCommand` thành ba nút một hàng, theo thứ tự đăng ký. Lệnh đăng ký bằng `OnAdminCommand` không được đưa vào, nên bàn phím này hiện cho ai cũng an toàn.

## Tự chọn lệnh

```go
// Thêm dấu "/" ở chỗ còn thiếu.
k := mtp.CommandKeyboard(2, "start", "help", "menu")
```

## Nhãn tuỳ ý

Nút không bắt buộc là lệnh. Nhãn thường sẽ đi tới handler `OnMessage`:

```go
k := mtp.NewKeyboard(
	mtp.KeyRow("📦 Đơn hàng", "👤 Tài khoản"),
	mtp.KeyRow("❓ Trợ giúp"),
)

bot.OnMessage(func(c *mtp.Context) error {
	switch c.Text() {
	case "📦 Đơn hàng":
		_, err := c.Send("Bạn chưa có đơn hàng nào.")
		return err
	}
	return nil
})
```

## Tuỳ biến

Ba method sau nối chuỗi được:

| Method | Tác dụng |
| --- | --- |
| `.OneTime()` | Ẩn bàn phím sau lần bấm đầu tiên |
| `.Persistent()` | Giữ bàn phím luôn mở thay vì thu về biểu tượng |
| `.Placeholder(text)` | Chữ gợi ý trong ô nhập khi còn trống |

```go
k := bot.CommandKeyboard(3).Placeholder("Chọn lệnh…")
```

## Ẩn bàn phím

Bàn phím ở lại cho tới khi bị thay hoặc bị gỡ:

```go
c.Send("Đã ẩn bàn phím.", mtp.RemoveKeyboard())
```

{: .note }
Bàn phím lệnh gửi tin vào chat; còn [nút inline](nut-inline.md) nằm dưới một tin cụ thể và không gửi gì vào chat. Một tin chỉ mang được một trong hai loại.
