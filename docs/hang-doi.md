---
title: Hàng đợi
layout: default
nav_order: 8
---

# Hàng đợi tin nhắn

Telegram cho một bot gửi khoảng 30 tin mỗi giây tính chung, và khoảng một tin mỗi giây vào cùng một chat. Vượt mức đó thì nhận lỗi `FLOOD_WAIT` và phải chờ. Hàng đợi giãn các tin ra để bot luôn ở dưới ngưỡng.

Dùng `SendMessage` cho câu trả lời tức thời trong handler. Dùng hàng đợi khi gửi nhiều tin liên tiếp, phát thông báo cho nhiều người, hoặc gửi từ tác vụ nền.

## Xếp tin vào hàng đợi

```go
for _, userID := range subscribers {
	bot.QueueMessage(userID, "📢 Có bản cập nhật mới!")
}
```

`QueueMessage` trả về ngay, không chờ tin được gửi. `QueuePhoto` làm việc tương tự cho ảnh. Cả hai nhận đủ các [tuỳ chọn gửi](gui-tin.md).

## Biết kết quả

Mỗi lần xếp hàng trả về một kênh nhận đúng một giá trị: `nil` nếu gửi được, hoặc lỗi.

```go
if err := <-bot.QueueMessage(chatID, text); err != nil {
	_, err := c.Reply("❌ Không gửi được: " + err.Error())
	return err
}
```

Kênh có bộ đệm, nên bỏ qua nó cũng không làm rò goroutine.

## Hàng đợi cư xử thế nào

- Một worker duy nhất gửi lần lượt, đúng thứ tự xếp hàng (FIFO).
- Hai lần gửi bất kỳ cách nhau ít nhất 50 ms; hai lần gửi vào cùng một chat cách nhau ít nhất 1 giây.
- Gặp `FLOOD_WAIT`, worker chờ đúng thời gian Telegram yêu cầu rồi gửi lại, tối đa ba lần cho mỗi tin. Trong lúc chờ, cả hàng đợi tạm dừng.
- Lỗi khác (chat không tồn tại, bot bị chặn…) được trả ngay qua kênh kết quả, không gửi lại.
- Hàng đợi đầy thì kênh kết quả nhận `mtp.ErrQueueFull` ngay lập tức; tin đó không được xếp.
- Khi bot dừng, các tin còn chờ nhận lỗi của context và không được gửi.

{: .warning }
Hàng đợi nằm trong bộ nhớ. Tin chưa gửi sẽ mất khi tiến trình dừng. Với thông báo quan trọng, hãy tự lưu danh sách cần gửi và đánh dấu từng tin sau khi kênh kết quả trả về `nil`.

## Tinh chỉnh

```go
bot, err := mtp.New(mtp.Config{
	// ...
	Queue: mtp.QueueConfig{
		Size:         5000,                   // mặc định 1024 tin chờ
		Interval:     40 * time.Millisecond,  // mặc định 50 ms
		ChatInterval: 1500 * time.Millisecond, // mặc định 1 giây
	},
})
```

Trường nào để giá trị không thì dùng mặc định.

## Việc tuỳ ý trong hàng đợi

`bot.Queue().Push` xếp một hàm bất kỳ, hưởng cùng nhịp giãn cách và cơ chế chờ `FLOOD_WAIT`:

```go
done := bot.Queue().Push(chatID, func(ctx context.Context) error {
	return bot.EditMessage(ctx, chatID, messageID, "Đã cập nhật")
})
```

`chatID` ở đây chỉ dùng để tính nhịp theo từng chat. `bot.Queue().Len()` cho số việc đang chờ.
