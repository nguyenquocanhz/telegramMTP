---
title: Gửi ảnh
layout: default
nav_order: 7
---

# Gửi ảnh

## Ba nguồn ảnh

| Hàm | Nguồn |
| --- | --- |
| `mtp.PhotoFile(path)` | Tệp trên đĩa |
| `mtp.PhotoURL(url)` | Địa chỉ http(s) công khai |
| `mtp.PhotoBytes(name, data)` | Dữ liệu trong bộ nhớ; `name` là tên tệp Telegram nhìn thấy, ví dụ `chart.png` |

## Gửi

```go
bot.OnCommand("logo", "Gửi logo", func(c *mtp.Context) error {
	_, err := c.SendPhoto(mtp.PhotoFile("assets/logo.png"), "Logo của chúng tôi")
	return err
})
```

| Method | Tác dụng |
| --- | --- |
| `c.SendPhoto(p, caption, opts...)` | Gửi ảnh vào chat hiện tại |
| `c.ReplyPhoto(p, caption, opts...)` | Gửi ảnh, trích dẫn tin đang xử lý |
| `bot.SendPhoto(ctx, chatID, p, caption, opts...)` | Gửi ảnh tới chat bất kỳ |
| `bot.QueuePhoto(chatID, p, caption, opts...)` | Gửi ảnh qua [hàng đợi](hang-doi.md) |

Tất cả trả về ID của tin mới (trừ `QueuePhoto`, trả về kênh kết quả). Chú thích để trống `""` thì ảnh không có chú thích.

## Chú thích và nút

Mọi `SendOption` đều áp dụng được cho ảnh. `mtp.HTML()` định dạng chú thích:

```go
c.SendPhoto(
	mtp.PhotoURL("https://example.com/banner.jpg"),
	"<b>Khuyến mãi</b> tới hết tuần",
	mtp.HTML(),
	mtp.WithInline(mtp.InlineRow(mtp.Btn("Xem ngay", "promo:open"))),
)
```

## Ảnh tạo trong bộ nhớ

Không cần ghi ra đĩa:

```go
var buf bytes.Buffer
if err := png.Encode(&buf, img); err != nil {
	return err
}
_, err := c.SendPhoto(mtp.PhotoBytes("chart.png", buf.Bytes()), "Biểu đồ hôm nay")
```

## Ảnh từ URL

Với `PhotoURL`, thư viện trước hết nhờ Telegram tự tải ảnh, nên dữ liệu không đi qua tiến trình của bạn. Nếu Telegram không tải hoặc không đọc được ảnh, thư viện tự tải về rồi đẩy lên thay.

{: .note }
Hiện mới có ảnh được bọc. Tài liệu, video và album gửi được qua `bot.API()` cùng gói `uploader` của gotd.
