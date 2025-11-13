# Blog Management System - Hướng dẫn khởi động nhanh

## Yêu cầu

- Go 1.24+
- PostgreSQL
- File `.env` (tùy chọn, có thể dùng giá trị mặc định)

## Khởi động nhanh

### 1. Cấu hình Database

Tạo file `.env` trong thư mục `examples` (hoặc sử dụng biến môi trường):

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=authkit
DB_SSLMODE=disable
```

### 2. Tạo Database

```bash
createdb authkit
```

### 3. Chạy ứng dụng

```bash
cd examples
go run .
```

Ứng dụng sẽ tự động:
- Kết nối database
- Chạy migrations (tạo bảng users, roles, rules, blogs)
- Khởi tạo roles mặc định (admin, editor, author, reader)
- Khởi tạo rules phân quyền
- Khởi động server trên port mặc định (8080)

### 4. Truy cập

- **Web UI**: http://localhost:8080
- **API**: http://localhost:8080/api

## API Endpoints chính

- `POST /api/auth/register` - Đăng ký tài khoản
- `POST /api/auth/login` - Đăng nhập
- `GET /api/blogs` - Xem danh sách blog (public)
- `POST /api/blogs` - Tạo blog (cần đăng nhập với role author/editor/admin)
- `GET /api/blogs/:id` - Xem chi tiết blog
- `PUT /api/blogs/:id` - Cập nhật blog
- `DELETE /api/blogs/:id` - Xóa blog

## Reset Database

Nếu bạn muốn xóa toàn bộ dữ liệu và chạy lại migrations từ đầu:

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

**⚠️ CẢNH BÁO**: Cả hai cách trên sẽ XÓA TẤT CẢ DỮ LIỆU trong database!

## Ghi chú

- Migrations chạy tự động khi khởi động
- Roles và rules được khởi tạo tự động lần đầu tiên
- Logs được lưu trong thư mục `logs/errors.log`

