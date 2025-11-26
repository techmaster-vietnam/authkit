#!/bin/bash

# Script tự động login và xóa role
# Sử dụng: ./delete_role.sh [role_id]

# Cấu hình mặc định
BASE_URL="${BASE_URL:-http://localhost:3000}"
EMAIL="${EMAIL:-admin@gmail.com}"
PASSWORD="${PASSWORD:-123456}"

# Tham số từ command line hoặc giá trị mặc định
ROLE_ID="${1:-6}"

# Màu sắc cho output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Hàm hiển thị thông báo
info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

# Kiểm tra jq có được cài đặt không
if ! command -v jq &> /dev/null; then
    error "jq chưa được cài đặt. Vui lòng cài đặt jq:"
    echo "  macOS: brew install jq"
    echo "  Linux: sudo apt-get install jq"
    exit 1
fi

# Kiểm tra curl có được cài đặt không
if ! command -v curl &> /dev/null; then
    error "curl chưa được cài đặt."
    exit 1
fi

# Bước 1: Login
info "Đang đăng nhập với email: $EMAIL..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${EMAIL}\",
    \"password\": \"${PASSWORD}\"
  }")

# Kiểm tra lỗi login
if echo "$LOGIN_RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    error "Lỗi đăng nhập:"
    echo "$LOGIN_RESPONSE" | jq '.'
    exit 1
fi

# Kiểm tra response có data không
if ! echo "$LOGIN_RESPONSE" | jq -e '.data' > /dev/null 2>&1; then
    error "Response không hợp lệ:"
    echo "$LOGIN_RESPONSE" | jq '.'
    exit 1
fi

# Lấy token
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    error "Không thể lấy token từ response:"
    echo "$LOGIN_RESPONSE" | jq '.'
    exit 1
fi

success "Đăng nhập thành công!"
info "Token: ${TOKEN:0:50}..."

# Lấy thông tin user
USER_EMAIL=$(echo "$LOGIN_RESPONSE" | jq -r '.data.user.email // "N/A"')
USER_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.data.user.id // "N/A"')
info "User ID: $USER_ID, Email: $USER_EMAIL"

# Bước 2: Xóa role
echo ""
info "Đang xóa role..."
info "  - Role ID: $ROLE_ID"

DELETE_ROLE_RESPONSE=$(curl -s -X DELETE "${BASE_URL}/api/roles/${ROLE_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN")

# Hiển thị response đẹp
echo ""
info "Response từ server:"
echo "$DELETE_ROLE_RESPONSE" | jq '.'

# Kiểm tra lỗi
if echo "$DELETE_ROLE_RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    error "Lỗi khi xóa role"
    
    # Xử lý error có thể là string hoặc object
    ERROR_TYPE=$(echo "$DELETE_ROLE_RESPONSE" | jq -r '.type // "UNKNOWN"')
    ERROR_VALUE=$(echo "$DELETE_ROLE_RESPONSE" | jq -r '.error')
    
    # Kiểm tra xem error là string hay object
    if echo "$DELETE_ROLE_RESPONSE" | jq -e '.error | type == "object"' > /dev/null 2>&1; then
        # Error là object, thử lấy message
        ERROR_MSG=$(echo "$DELETE_ROLE_RESPONSE" | jq -r '.error.message // .error | tostring')
    else
        # Error là string
        ERROR_MSG="$ERROR_VALUE"
    fi
    
    error "Loại lỗi: $ERROR_TYPE"
    error "Chi tiết: $ERROR_MSG"
    
    # Hiển thị thêm thông tin nếu có
    if echo "$DELETE_ROLE_RESPONSE" | jq -e '.data' > /dev/null 2>&1; then
        ERROR_DATA=$(echo "$DELETE_ROLE_RESPONSE" | jq '.data')
        info "Thông tin thêm:"
        echo "$ERROR_DATA" | jq '.'
    fi
    
    exit 1
fi

# Kiểm tra thành công
if echo "$DELETE_ROLE_RESPONSE" | jq -e '.message' > /dev/null 2>&1; then
    MESSAGE=$(echo "$DELETE_ROLE_RESPONSE" | jq -r '.message')
    success "$MESSAGE"
    echo ""
    info "Role ID $ROLE_ID đã được xóa thành công!"
    info "Stored procedure đã tự động:"
    info "  - Xóa tất cả bản ghi trong user_roles có role_id = $ROLE_ID"
    info "  - Xóa role_id khỏi mảng roles trong bảng rules"
    info "  - Xóa bản ghi trong bảng roles"
else
    # Kiểm tra HTTP status code
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${BASE_URL}/api/roles/${ROLE_ID}" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN")
    
    if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
        success "Xóa role thành công (HTTP $HTTP_CODE)"
    else
        error "Response không chứa message, có thể có lỗi (HTTP $HTTP_CODE)"
        exit 1
    fi
fi

