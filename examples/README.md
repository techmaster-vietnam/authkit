# Blog Management System - Hướng dẫn chạy thử ứng dụng mẫu

AuthKit đi kèm với một ứng dụng mẫu đầy đủ trong thư mục `examples` để bạn có thể nhanh chóng trải nghiệm các tính năng. Ứng dụng mẫu này là một Blog Management System với đầy đủ authentication và authorization.

## Yêu cầu hệ thống

- **Go**: 1.24+ (khuyến nghị Go 1.25+)
- **PostgreSQL**: 12+ (đã cài đặt và đang chạy)
- **VSCode** hoặc **Cursor IDE** với extension Go (khuyến nghị: [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go))

## Cấu hình Database

### Bước 1: Tạo database PostgreSQL

```bash
createdb authkit
```

### Bước 2: Tạo file `.env`

Tạo file `.env` trong thư mục `examples` với nội dung:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=authkit
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your-secret-key-change-in-production-use-long-random-string
JWT_EXPIRATION_HOURS=24

# Server Configuration
PORT=8080
READ_TIMEOUT_SECONDS=10
WRITE_TIMEOUT_SECONDS=10
```

**Lưu ý:** Thay đổi `DB_PASSWORD` và `JWT_SECRET` theo cấu hình PostgreSQL của bạn.

## Chạy ứng dụng từ Terminal

### Cách 1: Chạy trực tiếp

```bash
cd examples
go run .
```

### Cách 2: Build và chạy

```bash
cd examples
go build -o examples .
./examples
```

Ứng dụng sẽ tự động:
- ✅ Kết nối database
- ✅ Chạy migrations (tạo các bảng: users, roles, user_roles, rules)
- ✅ Seed dữ liệu mẫu (roles: admin, editor, author, reader)
- ✅ Khởi động server trên port 8080 (hoặc port trong `.env`)

**Truy cập ứng dụng:**
- 🌐 **Web UI**: http://localhost:8080
- 🔌 **API Base URL**: http://localhost:8080/api

## Chạy Debug với VSCode/Cursor IDE

### Cài đặt Go Extension

Đảm bảo bạn đã cài đặt Go extension trong VSCode/Cursor:
- Mở Extensions (Cmd+Shift+X / Ctrl+Shift+X)
- Tìm "Go" và cài đặt extension từ Google

### Cấu hình Debug

File `.vscode/launch.json` đã được tạo sẵn trong thư mục gốc của project với các configurations:

1. **Debug Examples App**: Sử dụng file `.env` từ thư mục `examples`
2. **Debug Examples App (with RESET_DB)**: Tự động reset database khi chạy (⚠️ XÓA TẤT CẢ DỮ LIỆU)
3. **Debug Examples App (Manual Env)**: Hardcode environment variables trong `launch.json`

**Lưu ý:** 
- File `launch.json` đã được cấu hình sẵn, bạn chỉ cần chỉnh sửa các giá trị trong `env` nếu cần
- Configuration thứ 2 (`RESET_DB=true`) sẽ tự động reset database khi chạy

### Cách sử dụng Debug

**Bước 1:** Mở file `examples/main.go` trong editor

**Bước 2:** Đặt breakpoint bằng cách click vào bên trái số dòng (hoặc nhấn F9)

**Bước 3:** Mở Debug panel:
- Nhấn `Cmd+Shift+D` (Mac) hoặc `Ctrl+Shift+D` (Windows/Linux)
- Hoặc click vào icon Debug ở sidebar

**Bước 4:** Chọn configuration "Debug Examples App" từ dropdown

**Bước 5:** Nhấn F5 hoặc click nút "Start Debugging" (▶️)

**Bước 6:** Ứng dụng sẽ chạy và dừng tại các breakpoint bạn đã đặt

**Các phím tắt Debug:**
- **F5**: Continue (tiếp tục chạy)
- **F10**: Step Over (chạy từng dòng, không vào hàm)
- **F11**: Step Into (chạy từng dòng, vào trong hàm)
- **Shift+F11**: Step Out (thoát khỏi hàm hiện tại)
- **Shift+F5**: Stop (dừng debug)

### Debug với file .env

File `launch.json` đã được cấu hình để sử dụng file `.env` từ thư mục `examples`:

```json
{
    "name": "Debug Examples App",
    "type": "go",
    "request": "launch",
    "mode": "auto",
    "program": "${workspaceFolder}/examples",
    "cwd": "${workspaceFolder}/examples",
    "envFile": "${workspaceFolder}/examples/.env",
    ...
}
```

Bạn chỉ cần tạo file `.env` trong thư mục `examples` và chạy debug như bình thường.

## Test API với ứng dụng mẫu

Sau khi ứng dụng đã chạy, bạn có thể test các API endpoints:

### Đăng ký tài khoản

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "123456",
    "full_name": "Test User"
  }'
```

### Đăng nhập

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "123456"
  }'
```

Response sẽ chứa `token` - sử dụng token này cho các request tiếp theo.

### Lấy thông tin profile (cần token)

```bash
curl -X GET http://localhost:8080/api/auth/profile \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### Tạo blog (cần đăng nhập với role author/editor/admin)

```bash
curl -X POST http://localhost:8080/api/blogs \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Blog",
    "content": "This is my first blog post"
  }'
```

## API Endpoints chính

- `POST /api/auth/register` - Đăng ký tài khoản
- `POST /api/auth/login` - Đăng nhập
- `GET /api/auth/profile` - Lấy thông tin profile (cần đăng nhập)
- `PUT /api/auth/profile` - Cập nhật profile (cần đăng nhập)
- `GET /api/blogs` - Xem danh sách blog (public)
- `POST /api/blogs` - Tạo blog (cần đăng nhập với role author/editor/admin)
- `GET /api/blogs/:id` - Xem chi tiết blog
- `PUT /api/blogs/:id` - Cập nhật blog
- `DELETE /api/blogs/:id` - Xóa blog

## Reset Database

Nếu bạn muốn reset database về trạng thái ban đầu:

### Cách 1: Sử dụng SQL (Khuyến nghị)

Kết nối vào PostgreSQL và chạy lệnh sau để xóa tất cả bảng:

```sql
DROP TABLE IF EXISTS blogs CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS rules CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;
```

Sau đó chạy lại ứng dụng, migrations sẽ tự động chạy:

```bash
cd examples
go run .
```

### Cách 2: Sử dụng biến môi trường RESET_DB

Ứng dụng có hỗ trợ tự động reset database khi set biến môi trường `RESET_DB=true`:

```bash
cd examples
RESET_DB=true go run .
```

Hoặc sử dụng configuration debug "Debug Examples App (with RESET_DB)" trong VSCode/Cursor.

**⚠️ CẢNH BÁO**: Cả hai cách trên sẽ XÓA TẤT CẢ DỮ LIỆU trong database!

## Xem Logs

Logs được lưu trong thư mục `examples/logs/errors.log`. Bạn có thể mở file này để xem chi tiết các lỗi và thông tin debug.

## Troubleshooting

### Vấn đề: Không kết nối được database

- Kiểm tra PostgreSQL đã chạy chưa: `pg_isready`
- Kiểm tra thông tin kết nối trong `.env` hoặc `launch.json`
- Kiểm tra database `authkit` đã được tạo chưa: `psql -l | grep authkit`

### Vấn đề: Port đã được sử dụng

- Thay đổi `PORT` trong `.env` hoặc `launch.json`
- Hoặc kill process đang sử dụng port: `lsof -ti:8080 | xargs kill -9`

### Vấn đề: Debug không hoạt động

- Đảm bảo đã cài Go extension
- Kiểm tra Go đã được cài đặt: `go version`
- Kiểm tra `launch.json` có đúng cấu hình không (file đã được tạo sẵn trong `.vscode/launch.json`)
- Thử restart VSCode/Cursor

### Vấn đề: Migrations lỗi

- Xóa database và tạo lại: `dropdb authkit && createdb authkit`
- Hoặc reset database bằng cách trên

## Ghi chú

- Migrations chạy tự động khi khởi động
- Roles và rules được khởi tạo tự động lần đầu tiên
- Logs được lưu trong thư mục `logs/errors.log`
- File `.vscode/launch.json` đã được cấu hình sẵn cho debug
