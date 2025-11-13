# Changelog

## [Unreleased] - 2024

### Cải tiến Bảo mật

#### 1. Chỉ PUBLIC cho phép Anonymous
- **Thay đổi**: Tất cả rule khác PUBLIC đều yêu cầu authentication
- **Lý do**: Đảm bảo không có endpoint nào vô tình cho phép anonymous truy cập
- **Ảnh hưởng**: Anonymous users sẽ bị từ chối ngay lập tức với lỗi 401 Unauthorized cho tất cả endpoint không phải PUBLIC

#### 2. Xử lý xung đột FORBIDE vs ALLOW
- **Thay đổi**: FORBIDE có ưu tiên cao hơn ALLOW khi có nhiều rule match cùng một endpoint
- **Logic**: 
  - Tìm tất cả rules match cho endpoint
  - Kiểm tra FORBIDE trước → Nếu user có role bị cấm → Từ chối
  - Nếu không có FORBIDE match → Kiểm tra ALLOW/AUTHENTICATED → Cho phép nếu match
- **Lý do**: Đảm bảo quy tắc cấm luôn được áp dụng, ngay cả khi có quy tắc cho phép

#### 3. Optional Role Context Header
- **Thêm**: Hỗ trợ header `X-Role-Context` (optional)
- **Mục đích**: Cho phép user chỉ định role cụ thể để sử dụng khi cần
- **Use case**: 
  - Audit trail: Xác định rõ role được sử dụng
  - Compliance: Một số quy định yêu cầu xác định role
  - Testing: Test với role cụ thể
- **Lưu ý**: Header này là optional, không bắt buộc. Nếu được cung cấp, user phải có role đó.

### Thay đổi API

Không có breaking changes. Tất cả các thay đổi đều backward compatible.

### Migration Guide

Không cần migration. Các rule hiện tại vẫn hoạt động bình thường. Tuy nhiên, bạn nên:

1. **Kiểm tra lại rules**: Đảm bảo các endpoint cần anonymous access đều có rule type PUBLIC
2. **Cập nhật documentation**: Thông báo cho team về quy tắc mới
3. **Test**: Kiểm tra lại các endpoint để đảm bảo hoạt động đúng

### Ví dụ

#### Trước đây (có thể cho phép anonymous nếu không cẩn thận):
```json
{
  "method": "GET",
  "path": "/api/data",
  "type": "ALLOW",
  "roles": []
}
```
→ Có thể cho phép anonymous nếu không kiểm tra authentication đúng cách

#### Bây giờ:
```json
{
  "method": "GET",
  "path": "/api/data",
  "type": "PUBLIC",
  "roles": []
}
```
→ Cho phép anonymous

```json
{
  "method": "GET",
  "path": "/api/data",
  "type": "ALLOW",
  "roles": []
}
```
→ Yêu cầu authentication (không cho phép anonymous)

### Xử lý xung đột

#### Ví dụ: User có cả role "editor" và "guest"

**Rule 1**: FORBIDE role "guest" cho endpoint `/api/admin/users`
**Rule 2**: ALLOW role "editor" cho endpoint `/api/admin/users`

**Kết quả**: User bị từ chối (FORBIDE ưu tiên hơn ALLOW)

**Giải pháp**: User có thể sử dụng header `X-Role-Context: editor` để chỉ sử dụng role editor, bypass rule FORBIDE cho role guest.

