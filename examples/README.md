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

## Ghi chú

- Migrations chạy tự động khi khởi động
- Roles và rules được khởi tạo tự động lần đầu tiên
- Logs được lưu trong thư mục `logs/errors.log`

