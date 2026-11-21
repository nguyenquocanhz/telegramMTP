---
title: Admin và danh sách lệnh
layout: default
nav_order: 9
---

# Admin và danh sách lệnh

## Khai báo admin

Admin là những user ID Telegram liệt kê trong `Config.AdminIDs`:

```go
bot, err := mtp.New(mtp.Config{
	// ...
	AdminIDs: []int64{111111111, 222222222},
})
```

Thường thì danh sách này đọc từ biến `ADMIN_ID`, các ID cách nhau bằng dấu phẩy hoặc khoảng trắng:

```go
admins, err := secrets.Int64List("ADMIN_ID") // ADMIN_ID=111111111,222222222
```

### Tìm ID của mình

Thêm một lệnh in ra ID, chạy bot, nhắn lệnh đó rồi đặt số nhận được vào `ADMIN_ID` và khởi động lại:

```go
bot.OnCommand("id", "Xem ID của bạn", func(c *mtp.Context) error {
	_, err := c.Reply(fmt.Sprintf("User ID: %d\nChat ID: %d", c.SenderID(), c.ChatID()))
	return err
})
```

Bot mẫu `natterbot` có sẵn lệnh `/id` này.

## Lệnh chỉ admin chạy được

```go
bot.OnAdminCommand("stats", "Trạng thái bot", func(c *mtp.Context) error {
	_, err := c.Reply(fmt.Sprintf("Hàng đợi: %d tin", bot.Queue().Len()))
	return err
})
```

Với người không phải admin, lệnh này cư xử **y hệt một lệnh không tồn tại**: tin đi tới `OnMessage`, không có thông báo "bạn không có quyền". Nhờ vậy người ngoài không dò ra được bot có những lệnh admin nào.

Kiểm tra quyền ở chỗ khác bằng `bot.IsAdmin(userID)`:

```go
if bot.IsAdmin(c.SenderID()) {
	text += "\n⭐ Bạn là admin."
}
```

`bot.AdminIDs()` trả về bản sao của danh sách admin.

## Báo tin cho admin

```go
bot.OnStart(func(ctx context.Context) error {
	bot.NotifyAdmins("✅ Bot đã khởi động")
	return nil
})
```

`NotifyAdmins` xếp tin vào [hàng đợi](hang-doi.md) cho từng admin và nhận các tuỳ chọn gửi như thường. Gửi thất bại chỉ được ghi log chứ không trả về lỗi.

{: .note }
Admin nào chưa từng bấm Start với bot thì không nhận được thông báo, vì Telegram không cho bot nhắn trước.

## Danh sách lệnh

Danh sách lệnh là menu hiện ra khi người dùng gõ `/` trong chat. Thư viện tự dựng nó từ các lệnh đã đăng ký và đẩy lên Telegram **mỗi lần bot khởi động**, nên bạn không phải khai báo với @BotFather.

| Ai | Thấy gì trong menu |
| --- | --- |
| Mọi người | Các lệnh `OnCommand` có mô tả |
| Từng admin | Như trên, thêm các lệnh `OnAdminCommand` có mô tả |

Quy tắc:

- Thứ tự trong menu là thứ tự đăng ký.
- Lệnh có mô tả rỗng vẫn chạy nhưng không hiện trong menu.
- Menu riêng của admin được đặt cho từng ID trong `AdminIDs`. Admin nào chưa mở chat với bot thì chưa có menu riêng (log ghi một cảnh báo); lệnh admin vẫn chạy được khi họ gõ tay.
- Đồng bộ thất bại không dừng bot, chỉ ghi cảnh báo.

Đăng ký thêm lệnh sau khi bot đã chạy thì gọi lại:

```go
err := bot.SyncCommands(ctx)
```

### Tự viết lệnh /help

`bot.Commands()` trả về danh sách lệnh theo thứ tự đăng ký, mỗi mục có `Name`, `Description` và `AdminOnly`:

```go
bot.OnCommand("help", "Danh sách lệnh", func(c *mtp.Context) error {
	var b strings.Builder
	for _, cmd := range bot.Commands() {
		if cmd.Description == "" || (cmd.AdminOnly && !bot.IsAdmin(c.SenderID())) {
			continue
		}
		fmt.Fprintf(&b, "/%s — %s\n", cmd.Name, cmd.Description)
	}
	_, err := c.Send(b.String())
	return err
})
```
