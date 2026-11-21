---
title: Sơ đồ luồng
layout: default
nav_order: 11
---

# Sơ đồ luồng

Ba giai đoạn trong vòng đời của bot: khởi động, nhận update, và gửi đi.

## 1. Khởi động

```mermaid
flowchart TD
    A["Chương trình bắt đầu"] --> B["SecretManager tìm cấu hình<br/>env → KEY_FILE → /run/secrets → secrets.enc → .env"]
    B --> C{"Đủ API_ID, API_HASH,<br/>BOT_TOKEN?"}
    C -- "thiếu" --> X["Thoát, báo rõ những nơi đã tìm"]
    C -- "đủ" --> D["mtp.New: kiểm tra cấu hình<br/>đăng ký handler"]
    D --> E["bot.Run: kết nối MTProto"]
    E --> F{"Đã có session.json?"}
    F -- "chưa" --> G["Đăng nhập bằng BOT_TOKEN<br/>rồi lưu session.json"]
    F -- "có" --> H["SyncCommands: đặt danh sách lệnh<br/>và menu riêng cho từng admin"]
    G --> H
    H --> I["Chạy các hàm OnStart"]
    I --> J["Chờ update cho tới khi ctx bị huỷ"]
```

Hàng đợi gửi bắt đầu chạy ngay sau khi đăng nhập, trước `SyncCommands`. Đồng bộ danh sách lệnh thất bại chỉ ghi cảnh báo; hàm `OnStart` trả về lỗi thì dừng bot.

## 2. Nhận update

```mermaid
flowchart TD
    U["Telegram gửi update"] --> V["Ghi nhớ access hash của user và chat"]
    V --> W{"Loại update"}
    W -- "bấm nút inline" --> CB["OnCallback<br/>khớp tiền tố dài nhất"]
    W -- "tin nhắn" --> O{"Tin do bot gửi?"}
    O -- "phải" --> Z["Bỏ qua"]
    O -- "không" --> R{"Bắt đầu bằng / và<br/>là lệnh đã đăng ký?"}
    R -- "không" --> T["Các handler OnMessage"]
    R -- "có" --> AD{"Lệnh admin?"}
    AD -- "không" --> HC["Handler của lệnh"]
    AD -- "có, người gửi là admin" --> HC
    AD -- "có, người gửi khác" --> T
    CB --> ACK["Tự xác nhận lần bấm<br/>nếu handler chưa Answer"]
```

Mỗi update chạy trong một goroutine riêng. Lỗi và panic của handler được ghi log, không làm bot dừng. Trong nhóm, lệnh gửi đích danh bot khác (`/ping@botkhac`) bị bỏ qua trước khi tới bước định tuyến.

## 3. Gửi đi

```mermaid
flowchart TD
    S1["Send · Reply · SendPhoto · Edit"] --> P["Đổi chat ID thành input peer<br/>bằng access hash đã ghi nhớ"]
    Q1["QueueMessage · QueuePhoto · Queue.Push"] --> Q2{"Hàng đợi còn chỗ?"}
    Q2 -- "hết" --> QF["Trả về ErrQueueFull"]
    Q2 -- "còn" --> Q3["Chờ tới lượt (FIFO)"]
    Q3 --> Q4["Giãn cách: 50 ms tính chung<br/>1 giây cho mỗi chat"]
    Q4 --> P
    P --> API["Gọi API MTProto"]
    API --> OK{"Kết quả"}
    OK -- "thành công" --> TG["Tin tới người dùng"]
    OK -- "FLOOD_WAIT, đang ở hàng đợi" --> FW["Chờ đúng thời gian yêu cầu<br/>gửi lại, tối đa 3 lần"]
    FW --> API
    OK -- "lỗi khác" --> ER["Trả lỗi cho nơi gọi"]
```

Gửi trực tiếp (`Send`, `Reply`…) không tự chờ `FLOOD_WAIT`: lỗi được trả thẳng về handler. Chỉ đường đi qua hàng đợi mới tự chờ và gửi lại.

## Toàn cảnh

```mermaid
flowchart LR
    ENV["Cấu hình<br/>env · .env · secrets"] --> SM["secret.Manager"]
    SM --> CL["mtp.Client"]
    TG(("Telegram")) -- "update" --> CL
    CL --> RT["Bộ định tuyến"]
    RT --> H["Handler của bạn"]
    H -- "gửi ngay" --> SND["Gửi"]
    H -- "xếp hàng" --> Q["Hàng đợi"]
    Q --> SND
    SND -- "MTProto" --> TG
    CL -. "khoá phiên" .-> SES[("session.json")]
```
