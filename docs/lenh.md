---
title: Lệnh và handler
layout: default
nav_order: 4
---

# Lệnh và handler

## Đăng ký lệnh

```go
bot.OnCommand("echo", "Nhắc lại nội dung", func(c *mtp.Context) error {
	if c.Args == "" {
		_, err := c.Reply("Cách dùng: /echo <nội dung>")
		return err
	}
	_, err := c.Reply(c.Args)
	return err
})
```

Tham số thứ hai là mô tả hiện trong menu `/` của Telegram. Để trống thì lệnh vẫn chạy nhưng không hiện trong menu. Đăng ký hai lần cùng một tên thì handler sau thay handler trước.

Handler có kiểu `func(c *mtp.Context) error`.

## Context

| Thành phần | Ý nghĩa |
| --- | --- |
| `c.Command` | Tên lệnh khớp được, không có dấu `/`. Rỗng với tin thường |
| `c.Args` | Phần còn lại của dòng sau tên lệnh |
| `c.ArgList()` | `Args` tách theo khoảng trắng |
| `c.ChatID()` | Chat nơi update xảy ra |
| `c.SenderID()` | Người gây ra update |
| `c.Text()` | Nội dung tin đến |
| `c.Message` | Tin đến: `ID`, `ChatID`, `SenderID`, `Text`, `ReplyToID`, `Date`, `Raw`. Là `nil` trong handler của nút inline |
| `c.Callback` | Lần bấm nút inline. Là `nil` trong handler của tin nhắn |
| `c.Client` | Client đang chạy |

`Context` nhúng `context.Context` của update, nên truyền thẳng `c` cho mọi hàm cần `ctx`. `c.Message.Raw` là `*tg.Message` đầy đủ của gotd, dành cho những trường chưa được đưa ra ngoài.

## Tin không phải lệnh

```go
bot.OnMessage(func(c *mtp.Context) error {
	if c.ChatID() <= 0 || c.Text() == "" {
		return nil // bỏ qua nhóm và tin không có chữ
	}
	_, err := c.Reply("Tôi chưa hiểu. Gõ /help để xem các lệnh.")
	return err
})
```

Có thể đăng ký nhiều handler `OnMessage`; chúng chạy theo thứ tự đăng ký và dừng ở handler đầu tiên trả về lỗi.

## Quy tắc định tuyến

- Tên lệnh không phân biệt hoa thường: `/Start` và `/start` là một.
- Trong nhóm, `/ping@botkhac` bị bỏ qua hoàn toàn; `/ping@botcuaban` và `/ping` đều khớp.
- Lệnh chưa đăng ký được coi là tin thường và đi tới `OnMessage`.
- Lệnh admin do người không phải admin gửi cũng đi tới `OnMessage`, y như một lệnh không tồn tại. Xem [Admin](admin.md).
- Tin do chính bot gửi đi không kích hoạt handler nào.

## Lỗi và panic

Lỗi handler trả về được ghi vào log (`Config.Logger`, mặc định `slog.Default()`) kèm chat và tên lệnh; bot vẫn chạy tiếp. Panic trong handler cũng được bắt lại và ghi log cùng stack trace.

Thư viện không tự báo lỗi cho người dùng. Muốn họ biết thì tự gửi tin trước khi trả về:

```go
bot.OnCommand("report", "Tạo báo cáo", func(c *mtp.Context) error {
	if err := buildReport(c); err != nil {
		c.Reply("❌ Không tạo được báo cáo, thử lại sau.")
		return err
	}
	_, err := c.Reply("✅ Xong.")
	return err
})
```

## Chạy đồng thời

Mỗi update chạy trong một goroutine riêng, nên handler chậm không chặn update khác. Đổi lại, hai handler có thể chạy cùng lúc: dữ liệu dùng chung giữa các handler phải được bảo vệ bằng `sync.Mutex` hoặc tương đương.

## Chạy việc khi bot vừa đăng nhập

```go
bot.OnStart(func(ctx context.Context) error {
	bot.NotifyAdmins("✅ Bot đã khởi động")
	return nil
})
```

Hàm này chạy một lần sau khi đăng nhập xong. Trả về lỗi sẽ dừng bot.

## Gọi API chưa được bọc

`bot.API()` trả về `*tg.Client` của gotd, gọi được mọi method MTProto:

```go
full, err := bot.API().HelpGetConfig(ctx)
```

`bot.Username()` cho username của bot (rỗng trước khi `Run` đăng nhập xong).
