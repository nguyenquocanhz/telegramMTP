---
title: Triển khai bằng Docker
layout: default
nav_order: 12
---

# Triển khai bằng Docker

Repo kèm sẵn `Dockerfile` và `docker-compose.yml` cho bot mẫu `natterbot`. Bot của riêng bạn dùng lại được gần như nguyên vẹn: chỉ cần đổi đường dẫn build trong `Dockerfile`.

## Chạy

```powershell
Copy-Item .env.example .env
```

Điền bốn giá trị vào `.env`:

```
API_ID=1234567
API_HASH=0123456789abcdef0123456789abcdef
BOT_TOKEN=123456789:AA...
ADMIN_ID=111111111
```

```powershell
docker compose up -d --build
```

```powershell
docker compose logs -f natterbot
```

Compose từ chối khởi động nếu thiếu một trong bốn giá trị trên, và nói rõ thiếu giá trị nào.

## Biến môi trường

| Biến | Bắt buộc | Ý nghĩa |
| --- | --- | --- |
| `API_ID`, `API_HASH` | có | Từ <https://my.telegram.org/apps> |
| `BOT_TOKEN` | có | Từ @BotFather |
| `ADMIN_ID` | có với compose | User ID của admin, cách nhau bằng dấu phẩy. Xem [Admin](admin.md) |
| `LOG_LEVEL` | không | `debug`, `info` (mặc định), `warn`, `error` |
| `TZ` | không | Múi giờ, mặc định `Asia/Ho_Chi_Minh` |
| `VERSION` | không | Nhãn phiên bản gắn vào binary và tag của image |
| `SESSION_FILE` | không | Image đặt sẵn là `/data/session.json` |

Chạy binary trực tiếp (không qua compose) thì `ADMIN_ID` là tuỳ chọn: thiếu nó bot vẫn chạy, chỉ ghi một cảnh báo và không ai dùng được lệnh admin.

## Image được dựng thế nào

- **Hai tầng.** Tầng build dùng `golang:1.26-alpine` để biên dịch binary tĩnh (`CGO_ENABLED=0`). Tầng chạy là `gcr.io/distroless/static-debian12:nonroot`: có chứng chỉ CA và dữ liệu múi giờ, không có shell.
- **Không chạy bằng root.** Tiến trình chạy dưới user `nonroot`.
- **Bí mật không vào image.** `.dockerignore` loại `.env`, `secrets/`, `secrets.enc` và `session.json` khỏi build context.

## Compose bảo vệ container ra sao

| Thiết lập | Tác dụng |
| --- | --- |
| `read_only: true` | Hệ thống tệp chỉ đọc; chỉ volume `/data` ghi được |
| `cap_drop: [ALL]` | Bỏ hết Linux capability |
| `no-new-privileges` | Tiến trình không leo thang quyền được |
| Không có `ports` | Bot chỉ kết nối ra ngoài, không mở cổng nào |
| `memory: 256M` | Giới hạn bộ nhớ |
| `restart: unless-stopped` | Tự chạy lại sau sự cố hoặc khi máy khởi động lại |

## Tệp phiên

`session.json` chứa khoá đăng nhập MTProto và nằm trong volume `natterbot-data`. Nhờ đó khởi động lại hay nâng cấp image đều không phải đăng nhập lại.

{: .warning }
Ai có `session.json` là điều khiển được bot. Giữ nó như giữ mật khẩu, và đừng đưa vào image hay git.

Muốn bot đăng nhập lại từ đầu (ví dụ sau khi đổi bot token), xoá volume:

```powershell
docker compose down -v
```

## Dùng Docker secrets thay cho biến môi trường

[SecretManager](secret.md) tự đọc `/run/secrets/<tên khoá viết thường>`, nên không cần đổi mã:

1. Ghi token vào `secrets/bot_token.txt` (thư mục `secrets/` đã nằm trong `.gitignore`).
2. Trong `docker-compose.yml`, xoá dòng `BOT_TOKEN` ở phần `environment`.
3. Bỏ chú thích hai khối `secrets:` có sẵn trong file.

```yaml
services:
  natterbot:
    secrets:
      - bot_token

secrets:
  bot_token:
    file: ./secrets/bot_token.txt
```

## Nâng cấp

```powershell
git pull
```

```powershell
docker compose up -d --build
```

## Build không cần Docker

Binary tĩnh cho Linux, dựng từ Windows:

```powershell
$env:GOOS = "linux"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"; go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o bin/natterbot-linux-amd64 ./cmd/natterbot
```

Chép binary lên máy chủ, đặt các biến môi trường (hoặc file `.env` cùng thư mục) rồi chạy. `natterbot version` in ra nhãn phiên bản đã gắn lúc build.
