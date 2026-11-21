---
title: Tham chiếu API
layout: default
nav_order: 13
---

# Tham chiếu API
{: .no_toc }

Toàn bộ API công khai của hai package. Chú thích trong mã nguồn (xem bằng `go doc`) là nguồn chính xác nhất:

```powershell
go doc -all github.com/nguyenquocanhz/telegramMTP/mtp
```

## Mục lục
{: .no_toc .text-delta }

- TOC
{:toc}

---

## Package mtp

### Khởi tạo và chạy

| Khai báo | Mô tả |
| --- | --- |
| `func New(cfg Config) (*Client, error)` | Kiểm tra cấu hình, chuẩn bị client. Chưa kết nối mạng |
| `func (c *Client) Run(ctx context.Context) error` | Kết nối, đăng nhập, chặn tới khi `ctx` bị huỷ |
| `func (c *Client) OnStart(fn func(ctx context.Context) error)` | Chạy `fn` một lần sau khi đăng nhập; lỗi sẽ dừng bot |
| `func (c *Client) Username() string` | Username của bot; rỗng trước khi đăng nhập |
| `func (c *Client) API() *tg.Client` | Client gotd thô cho những lời gọi chưa được bọc |

```go
type Config struct {
	APIID       int          // từ my.telegram.org/apps
	APIHash     string
	BotToken    string       // từ @BotFather
	SessionFile string       // mặc định "session.json"
	AdminIDs    []int64      // ai được chạy lệnh OnAdminCommand
	Logger      *slog.Logger // mặc định slog.Default()
	Queue       QueueConfig  // giá trị không là đủ dùng
}
```

`New` trả về lỗi khi thiếu `APIID`, `APIHash` hoặc `BotToken`.

### Đăng ký handler

| Khai báo | Mô tả |
| --- | --- |
| `type Handler func(c *Context) error` | Xử lý một update; lỗi trả về được ghi log |
| `func (c *Client) OnCommand(name, description string, h Handler)` | Bắt lệnh `/name` |
| `func (c *Client) OnAdminCommand(name, description string, h Handler)` | Bắt lệnh chỉ admin chạy được |
| `func (c *Client) OnMessage(h Handler)` | Bắt tin không phải lệnh đã biết |
| `func (c *Client) OnCallback(prefix string, h Handler)` | Bắt nút inline có `data` bắt đầu bằng `prefix` |

### Context

```go
type Context struct {
	context.Context
	Client   *Client
	Message  *Message  // nil trong handler OnCallback
	Callback *Callback // nil trong handler tin nhắn
	Command  string    // tên lệnh, không có "/"
	Args     string    // phần còn lại của dòng
}
```

| Method | Trả về |
| --- | --- |
| `ChatID() int64` | Chat nơi update xảy ra |
| `SenderID() int64` | Người gây ra update |
| `Text() string` | Nội dung tin đến; rỗng nếu không có tin |
| `ArgList() []string` | `Args` tách theo khoảng trắng |
| `Send(text string, opts ...SendOption) (int, error)` | Gửi vào chat hiện tại |
| `Reply(text string, opts ...SendOption) (int, error)` | Gửi kèm trích dẫn tin đang xử lý |
| `SendPhoto(p Photo, caption string, opts ...SendOption) (int, error)` | Gửi ảnh vào chat hiện tại |
| `ReplyPhoto(p Photo, caption string, opts ...SendOption) (int, error)` | Gửi ảnh kèm trích dẫn |
| `Answer(text string) error` | Xác nhận lần bấm nút, kèm thông báo ngắn |
| `AnswerAlert(text string) error` | Xác nhận lần bấm nút bằng hộp thoại |
| `Edit(text string, opts ...SendOption) error` | Thay nội dung tin mang nút vừa bấm |

```go
type Message struct {
	ID        int
	ChatID    int64
	SenderID  int64
	Text      string
	ReplyToID int        // 0 nếu không trả lời tin nào
	Date      time.Time
	Raw       *tg.Message // tin MTProto đầy đủ
}

type Callback struct {
	QueryID   int64
	Data      string // chuỗi đã truyền cho Btn
	MessageID int    // tin mang nút vừa bấm
}
```

### Gửi

| Khai báo | Mô tả |
| --- | --- |
| `func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, opts ...SendOption) (int, error)` | Gửi tin, trả về ID tin mới |
| `func (c *Client) SendPhoto(ctx context.Context, chatID int64, p Photo, caption string, opts ...SendOption) (int, error)` | Gửi ảnh, trả về ID tin mới |
| `func (c *Client) EditMessage(ctx context.Context, chatID int64, messageID int, text string, opts ...SendOption) error` | Sửa tin bot đã gửi |

### Tuỳ chọn gửi

| Khai báo | Mô tả |
| --- | --- |
| `func HTML() SendOption` | Hiểu văn bản là HTML của Telegram |
| `func Silent() SendOption` | Không kèm âm báo |
| `func NoPreview() SendOption` | Tắt xem trước liên kết |
| `func ReplyTo(messageID int) SendOption` | Trả lời `messageID` |
| `func WithKeyboard(k *Keyboard) SendOption` | Gắn bàn phím lệnh |
| `func RemoveKeyboard() SendOption` | Ẩn bàn phím lệnh |
| `func WithInline(rows ...[]InlineButton) SendOption` | Gắn nút inline |

### Bàn phím lệnh

| Khai báo | Mô tả |
| --- | --- |
| `func NewKeyboard(rows ...[]string) *Keyboard` | Bàn phím từ các hàng nhãn |
| `func KeyRow(labels ...string) []string` | Một hàng nhãn |
| `func CommandKeyboard(perRow int, commands ...string) *Keyboard` | Xếp lệnh thành lưới, tự thêm `/` |
| `func (c *Client) CommandKeyboard(perRow int) *Keyboard` | Như trên, từ các lệnh `OnCommand` đã đăng ký |
| `func (k *Keyboard) OneTime() *Keyboard` | Ẩn sau lần bấm đầu |
| `func (k *Keyboard) Persistent() *Keyboard` | Luôn mở |
| `func (k *Keyboard) Placeholder(text string) *Keyboard` | Chữ gợi ý trong ô nhập |

### Nút inline

| Khai báo | Mô tả |
| --- | --- |
| `func Btn(text, data string) InlineButton` | Nút callback; `data` tối đa 64 byte |
| `func URLBtn(text, url string) InlineButton` | Nút mở liên kết |
| `func InlineRow(buttons ...InlineButton) []InlineButton` | Một hàng nút |

### Ảnh

| Khai báo | Mô tả |
| --- | --- |
| `func PhotoFile(path string) Photo` | Ảnh từ tệp trên đĩa |
| `func PhotoURL(url string) Photo` | Ảnh từ địa chỉ http(s) công khai |
| `func PhotoBytes(name string, data []byte) Photo` | Ảnh trong bộ nhớ |

### Hàng đợi

| Khai báo | Mô tả |
| --- | --- |
| `func (c *Client) QueueMessage(chatID int64, text string, opts ...SendOption) <-chan error` | Xếp một tin vào hàng đợi |
| `func (c *Client) QueuePhoto(chatID int64, p Photo, caption string, opts ...SendOption) <-chan error` | Xếp một ảnh vào hàng đợi |
| `func (c *Client) Queue() *Queue` | Hàng đợi của client |
| `func (q *Queue) Push(chatID int64, fn func(ctx context.Context) error) <-chan error` | Xếp một việc tuỳ ý |
| `func (q *Queue) Len() int` | Số việc đang chờ |
| `var ErrQueueFull` | Hàng đợi hết chỗ |

```go
type QueueConfig struct {
	Size         int           // mặc định DefaultQueueSize = 1024
	Interval     time.Duration // mặc định DefaultQueueInterval = 50 ms
	ChatInterval time.Duration // mặc định DefaultChatInterval = 1 giây
}
```

### Admin và danh sách lệnh

| Khai báo | Mô tả |
| --- | --- |
| `func (c *Client) IsAdmin(userID int64) bool` | `userID` có trong `Config.AdminIDs` không |
| `func (c *Client) AdminIDs() []int64` | Bản sao danh sách admin |
| `func (c *Client) NotifyAdmins(text string, opts ...SendOption)` | Xếp một tin cho mọi admin |
| `func (c *Client) Commands() []CommandInfo` | Lệnh đã đăng ký, theo thứ tự đăng ký |
| `func (c *Client) SyncCommands(ctx context.Context) error` | Đẩy danh sách lệnh lên Telegram |

```go
type CommandInfo struct {
	Name        string
	Description string
	AdminOnly   bool
}
```

---

## Package secret

### Manager

| Khai báo | Mô tả |
| --- | --- |
| `func New(opts Options) (*Manager, error)` | Tạo manager; chỉ lỗi khi một nguồn có mặt nhưng hỏng |
| `func (m *Manager) Get(key string) (string, bool)` | Giá trị và có tìm thấy không |
| `func (m *Manager) GetOr(key, fallback string) string` | Giá trị, hoặc `fallback` |
| `func (m *Manager) Require(key string) (string, error)` | Giá trị, hoặc lỗi liệt kê những nơi đã tìm |
| `func (m *Manager) Int64(key string) (int64, error)` | Bí mật bắt buộc dạng số |
| `func (m *Manager) Int64List(key string) ([]int64, error)` | Danh sách số tuỳ chọn, cách nhau bằng dấu phẩy hoặc khoảng trắng |
| `func (m *Manager) Source(key string) string` | Nơi tìm thấy khoá; an toàn để ghi log |
| `func (m *Manager) StorePath() string` | Đường dẫn kho mã hoá đang dùng |
| `func Mask(s string) string` | Che một bí mật để ghi log |

```go
type Options struct {
	DotEnv     string // mặc định ".env"
	SecretsDir string // mặc định "/run/secrets"
	StorePath  string // mặc định: biến SECRET_STORE, rồi "secrets.enc"
}
```

### Store

| Khai báo | Mô tả |
| --- | --- |
| `func OpenStore(path, passphrase string) (*Store, error)` | Giải mã kho; thiếu file thì cho kho rỗng |
| `func (s *Store) Get(key string) (string, bool)` | Đọc một giá trị |
| `func (s *Store) Set(key, value string)` | Ghi vào bộ nhớ; gọi `Save` để lưu |
| `func (s *Store) Delete(key string) bool` | Xoá khoá; cho biết khoá có tồn tại không |
| `func (s *Store) Keys() []string` | Các khoá, theo thứ tự chữ cái |
| `func (s *Store) Save() error` | Mã hoá và ghi file, thay thế nguyên tử |
| `var ErrWrongKey` | Sai khoá chủ hoặc file đã bị sửa |
| `const MasterKeyName = "SECRET_MASTER_KEY"` | Tên khoá chứa mật khẩu chủ |
