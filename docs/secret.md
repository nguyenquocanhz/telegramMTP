---
title: SecretManager
layout: default
nav_order: 10
---

# SecretManager

Package `secret` nạp thông tin đăng nhập mà không buộc mã nguồn phải biết chúng nằm ở đâu. Cùng một bản build chạy được với file `.env` trên máy dev, biến môi trường trên máy chủ, và Docker secret trong cụm.

```go
import "github.com/nguyenquocanhz/telegramMTP/secret"
```

Package này không phụ thuộc vào `mtp`, dùng riêng được.

## Thứ tự tìm

Một khoá, ví dụ `BOT_TOKEN`, được tìm lần lượt ở năm nơi, gặp đâu lấy đó:

| # | Nguồn | Ví dụ |
| --- | --- | --- |
| 1 | Biến môi trường | `BOT_TOKEN=123:AA...` |
| 2 | Tệp do biến `<KHOÁ>_FILE` trỏ tới | `BOT_TOKEN_FILE=/etc/bot/token` |
| 3 | Docker/Kubernetes secret | `/run/secrets/bot_token` (tên khoá viết thường) |
| 4 | Kho mã hoá | `secrets.enc`, cần `SECRET_MASTER_KEY` |
| 5 | File `.env` | dòng `BOT_TOKEN=...` |

Giá trị rỗng được coi là chưa đặt, nên một dòng `BOT_TOKEN=` bỏ trống trong file compose không che mất giá trị thật ở nguồn phía sau.

## Đọc bí mật

```go
secrets, err := secret.New(secret.Options{})
if err != nil {
	return err
}

token, err := secrets.Require("BOT_TOKEN")      // bắt buộc
apiID, err := secrets.Int64("API_ID")           // bắt buộc, là số
admins, err := secrets.Int64List("ADMIN_ID")    // tuỳ chọn: "111,222" hoặc "111 222"
level := secrets.GetOr("LOG_LEVEL", "info")     // tuỳ chọn, có mặc định
value, ok := secrets.Get("WEBHOOK_URL")         // tuỳ chọn
```

| Method | Khi khoá chưa đặt |
| --- | --- |
| `Get(key)` | trả về `"", false` |
| `GetOr(key, fallback)` | trả về `fallback` |
| `Require(key)` | lỗi, liệt kê đủ những nơi đã tìm |
| `Int64(key)` | lỗi như `Require`; cũng lỗi nếu giá trị không phải số |
| `Int64List(key)` | danh sách rỗng; lỗi nếu có phần tử không phải số |

`secret.New` chỉ thất bại khi một nguồn có mặt nhưng hỏng: file `.env` không đọc được, hoặc kho mã hoá không mở được bằng khoá chủ đã cho. Thiếu file thì không sao.

## Ghi log an toàn

Đừng bao giờ ghi giá trị bí mật vào log. Thay vào đó ghi **nguồn** của nó và một bản đã che:

```go
log.Info("configuration loaded",
	"bot_token_from", secrets.Source("BOT_TOKEN"), // "env", ".env", "file <path>", "store <path>"
	"bot_token", secret.Mask(token),               // giữ vài ký tự đầu cuối, đủ để phân biệt
)
```

## Kho mã hoá

Kho là một file khoá–giá trị mã hoá bằng AES-256-GCM. Khoá mã hoá được dẫn xuất từ mật khẩu chủ bằng PBKDF2-HMAC-SHA256 với 600 000 vòng. File này để cạnh binary hay đưa vào bản sao lưu đều được, miễn là mật khẩu chủ nằm ở chỗ khác.

Mật khẩu chủ lấy từ khoá `SECRET_MASTER_KEY`, tìm qua các nguồn 1, 2, 3 và 5 ở trên (không bao giờ lấy từ chính kho).

### Bằng dòng lệnh

`natterbot` kèm sẵn công cụ quản lý kho:

```powershell
$env:SECRET_MASTER_KEY = "<mật khẩu chủ>"
```

```powershell
.\bin\natterbot.exe secret set BOT_TOKEN
```

```powershell
.\bin\natterbot.exe secret list
```

```powershell
.\bin\natterbot.exe secret rm BOT_TOKEN
```

`secret set` đọc giá trị từ stdin, nên bí mật không lọt vào lịch sử shell. `secret list` chỉ in tên khoá và giá trị đã che.

### Bằng mã

```go
store, err := secret.OpenStore("secrets.enc", passphrase)
if errors.Is(err, secret.ErrWrongKey) {
	// sai mật khẩu chủ, hoặc file đã bị sửa
}
store.Set("BOT_TOKEN", token)
if err := store.Save(); err != nil {
	return err
}
```

`Save` ghi ra file tạm rồi đổi tên, nên sự cố giữa chừng không để lại file hỏng. Mỗi lần lưu dùng salt và nonce mới. `Get`, `Delete` và `Keys` làm đúng như tên gọi.

{: .warning }
Mất mật khẩu chủ là mất kho: không có cách khôi phục. AES-GCM cũng không phân biệt được "sai mật khẩu" với "file bị sửa", nên cả hai đều cho `ErrWrongKey`.

## Đổi vị trí mặc định

```go
secrets, err := secret.New(secret.Options{
	DotEnv:     "config/bot.env",  // mặc định ".env"
	SecretsDir: "/var/run/secrets", // mặc định "/run/secrets"
	StorePath:  "config/bot.enc",   // mặc định: biến SECRET_STORE, rồi "secrets.enc"
})
```

## Cú pháp .env

```
# dòng chú thích
API_ID=1234567
BOT_TOKEN="123456789:AA..."
ADMIN_ID=111111111,222222222
```

Mỗi dòng một cặp `KHOÁ=giá trị`; giá trị có thể bọc trong dấu nháy đơn hoặc kép, và tiền tố `export ` được bỏ qua. Dòng không đúng dạng làm `secret.New` báo lỗi kèm số dòng. Biến môi trường thật luôn thắng giá trị trong `.env`.
