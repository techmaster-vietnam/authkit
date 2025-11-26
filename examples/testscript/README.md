# Scripts Bash cho AuthKit

Thư mục này chứa các script bash tiện ích để tương tác với AuthKit API.

## Scripts có sẵn

### `create_role.sh` - Tạo role mới

Script tự động login và tạo role mới trong hệ thống.

#### Cài đặt dependencies

```bash
# macOS
brew install jq

# Linux (Ubuntu/Debian)
sudo apt-get install jq

# Linux (CentOS/RHEL)
sudo yum install jq
```

#### Cách sử dụng

**1. Sử dụng với giá trị mặc định:**
```bash
cd examples/bash
./create_role.sh
```
- Role ID: 5
- Role Name: moderator
- Is System: false

**2. Chỉ định role ID và name:**
```bash
./create_role.sh 10 "editor" false
```

**3. Sử dụng biến môi trường để cấu hình:**
```bash
export BASE_URL="http://localhost:3000"
export EMAIL="admin@example.com"
export PASSWORD="your_password"
./create_role.sh 5 "moderator" false
```

#### Tham số

- `$1` (optional): Role ID (mặc định: 5)
- `$2` (optional): Role Name (mặc định: "moderator")
- `$3` (optional): Is System (mặc định: false)

#### Biến môi trường

- `BASE_URL`: URL của API server (mặc định: http://localhost:3000)
- `EMAIL`: Email để đăng nhập (mặc định: admin@example.com)
- `PASSWORD`: Mật khẩu để đăng nhập (mặc định: 123456)

#### Ví dụ sử dụng

```bash
# Tạo role với ID 10, tên "author", không phải system role
./create_role.sh 10 "author" false

# Tạo role với ID 20, tên "super_editor", là system role
./create_role.sh 20 "super_editor" true

# Sử dụng với custom server và credentials
BASE_URL="http://api.example.com:8080" \
EMAIL="admin@company.com" \
PASSWORD="secure_password" \
./create_role.sh 15 "manager" false
```

#### Output

Script sẽ hiển thị:
- Thông tin đăng nhập (token preview, user info)
- Thông tin role đang tạo
- Response từ server
- Kết quả thành công hoặc lỗi

#### Lưu ý

- User phải có role "admin" để tạo role mới
- Role ID và name phải unique (không trùng với role đã tồn tại)
- Không được tạo role tên "super_admin" qua API
- Script yêu cầu `jq` và `curl` đã được cài đặt

---

### `delete_role.sh` - Xóa role

Script tự động login và xóa role khỏi hệ thống. Script sử dụng stored procedure để đảm bảo tính nhất quán dữ liệu:
- Xóa tất cả bản ghi trong bảng `user_roles` có `role_id` tương ứng
- Xóa `role_id` khỏi mảng `roles` trong bảng `rules`
- Xóa bản ghi trong bảng `roles`

#### Cài đặt dependencies

Tương tự như `create_role.sh`, cần cài đặt `jq` và `curl`:

```bash
# macOS
brew install jq

# Linux (Ubuntu/Debian)
sudo apt-get install jq

# Linux (CentOS/RHEL)
sudo yum install jq
```

#### Cách sử dụng

**1. Sử dụng với giá trị mặc định:**
```bash
cd examples/testscript
./delete_role.sh
```
- Role ID: 7 (mặc định)

**2. Chỉ định role ID:**
```bash
./delete_role.sh 10
```

**3. Sử dụng biến môi trường để cấu hình:**
```bash
export BASE_URL="http://localhost:3000"
export EMAIL="admin@example.com"
export PASSWORD="your_password"
./delete_role.sh 5
```

#### Tham số

- `$1` (optional): Role ID cần xóa (mặc định: 7)

#### Biến môi trường

- `BASE_URL`: URL của API server (mặc định: http://localhost:3000)
- `EMAIL`: Email để đăng nhập (mặc định: admin@gmail.com)
- `PASSWORD`: Mật khẩu để đăng nhập (mặc định: 123456)

#### Ví dụ sử dụng

```bash
# Xóa role có ID 10
./delete_role.sh 10

# Xóa role có ID 20
./delete_role.sh 20

# Sử dụng với custom server và credentials
BASE_URL="http://api.example.com:8080" \
EMAIL="admin@company.com" \
PASSWORD="secure_password" \
./delete_role.sh 15
```

#### Output

Script sẽ hiển thị:
- Thông tin đăng nhập (token preview, user info)
- Thông tin role đang xóa
- Response từ server
- Kết quả thành công hoặc lỗi
- Thông tin về các bước dọn dẹp dữ liệu được thực hiện bởi stored procedure

#### Lưu ý

- User phải có role "admin" để xóa role
- Không thể xóa system role (bao gồm "super_admin")
- Stored procedure sẽ tự động dọn dẹp dữ liệu liên quan:
  - Xóa tất cả bản ghi trong `user_roles` có `role_id` tương ứng
  - Xóa `role_id` khỏi mảng `roles` trong tất cả rules
  - Cuối cùng xóa bản ghi trong bảng `roles`
- Script yêu cầu `jq` và `curl` đã được cài đặt

