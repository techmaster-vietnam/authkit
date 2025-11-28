#!/usr/bin/env python3
"""Script test đăng ký user với các trường custom (mobile, address)
Kiểm tra validation và test login sau khi đăng ký thành công"""
import json
import sys
from typing import Dict, List, Optional, Tuple

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

from share import (
    info, success, error, get_base_url, print_section,
    get_user_detail, confirm_reset, login_account,
    login_safe, delete_user
)

# Định nghĩa cấu trúc user
UserData = Dict[str, str]

def register_user(user_data: UserData) -> Tuple[bool, Optional[Dict], Optional[str]]:
    """
    Đăng ký user mới
    
    Args:
        user_data: Dictionary chứa thông tin user (email, password, full_name, mobile, address)
    
    Returns:
        Tuple (success, user_info, error_message)
        - success: True nếu đăng ký thành công, False nếu lỗi
        - user_info: Thông tin user nếu thành công, None nếu lỗi
        - error_message: Thông báo lỗi nếu có, None nếu thành công
    """
    base_url = get_base_url()
    
    # Chuẩn bị request body
    request_body = {
        "email": user_data.get("email", ""),
        "password": user_data.get("password", ""),
        "full_name": user_data.get("full_name", ""),
    }
    
    # Thêm các trường custom nếu có
    if "mobile" in user_data:
        request_body["mobile"] = user_data["mobile"]
    if "address" in user_data:
        request_body["address"] = user_data["address"]
    
    try:
        info(f"Đang đăng ký user: {user_data.get('email', 'N/A')}...")
        resp = requests.post(
            f"{base_url}/api/auth/register",
            json=request_body,
            timeout=10
        )
        
        # Parse response
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, None, f"Response không phải JSON. Status: {resp.status_code}"
        
        # Kiểm tra lỗi
        if resp.status_code != 201:
            # Sử dụng handle_error_response để format error message
            error_msg = "Lỗi không xác định"
            
            # Thử lấy từ "error" object (nếu là dict)
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
            elif isinstance(error_obj, str):
                error_msg = error_obj
            
            # Thử lấy từ top level "message" (format của goerrorkit)
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            
            # Thử lấy chi tiết validation từ top level "data"
            error_details = {}
            if "data" in resp_data and isinstance(resp_data.get("data"), dict):
                error_details = resp_data.get("data", {})
            
            # Format error message
            if error_details:
                error_msg += f" | Chi tiết: {json.dumps(error_details, ensure_ascii=False)}"
            
            return False, None, error_msg
        
        # Kiểm tra response có data không
        if "data" not in resp_data:
            return False, None, "Response không chứa data"
        
        user_info = resp_data.get("data", {})
        success(f"Đăng ký thành công! User ID: {user_info.get('id', 'N/A')}")
        return True, user_info, None
        
    except requests.exceptions.RequestException as e:
        return False, None, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, None, f"Lỗi không xác định: {str(e)}"

def display_registered_users(token: str, user_ids: List[str]) -> None:
    """
    Hiển thị danh sách các user đã đăng ký thành công
    
    Args:
        token: JWT token để xác thực
        user_ids: Danh sách user IDs cần hiển thị
    """
    print_section("DANH SÁCH USER ĐÃ ĐĂNG KÝ THÀNH CÔNG")
    
    if not user_ids:
        info("Không có user nào được đăng ký thành công.")
        print()
        return
    
    print(f"Tổng số: {len(user_ids)} user(s)")
    print()
    print("-" * 80)
    
    for idx, user_id in enumerate(user_ids, 1):
        print(f"\n[{idx}/{len(user_ids)}] User ID: {user_id}")
        user_detail = get_user_detail(token, user_id, verbose=False)
        
        if user_detail:
            user = user_detail.get("user", {})
            print(f"   Email: {user.get('email', 'N/A')}")
            print(f"   Full Name: {user.get('full_name', 'N/A')}")
            print(f"   Mobile: {user.get('mobile', 'N/A')}")
            print(f"   Address: {user.get('address', 'N/A')}")
            print(f"   Is Active: {user.get('is_active', 'N/A')}")
        else:
            error(f"   Không thể lấy thông tin user ID: {user_id}")
        
        print("-" * 80)
    
    print()

def main():
    """Hàm main để test đăng ký user"""
    
    # Mảng các user để test
    # Một số bản ghi có dữ liệu không hợp lệ để test validation
    test_users: List[UserData] = [
        # Test case 1: User hợp lệ đầy đủ thông tin
        {
            "email": "test1@example.com",
            "password": "Abc1234@-",
            "full_name": "Test User 1",
            "mobile": "0901234567",
            "address": "123 Main Street, Ho Chi Minh City"
        },
        # Test case 2: User hợp lệ không có mobile và address
        {
            "email": "test2@example.com",
            "password": "Password123@-",
            "full_name": "Test User 2"
        },
        # Test case 3: Email không hợp lệ (thiếu @)
        {
            "email": "invalidemail.com",
            "password": "123456",
            "full_name": "Test User 3",
            "mobile": "0901234567",
            "address": "456 Test Avenue"
        },
        # Test case 4: Password quá ngắn (< 6 ký tự)
        {
            "email": "test4@example.com",
            "password": "12345",
            "full_name": "Test User 4",
            "mobile": "0901234567",
            "address": "789 Test Road"
        },
        # Test case 5: Email trống
        {
            "email": "",
            "password": "123456",
            "full_name": "Test User 5",
            "mobile": "0901234567",
            "address": "321 Test Lane"
        },
        # Test case 6: Password trống
        {
            "email": "test6@example.com",
            "password": "",
            "full_name": "Test User 6",
            "mobile": "0901234567",
            "address": "654 Test Boulevard"
        },
        # Test case 7: User hợp lệ với mobile và address
        {
            "email": "test7@example.com",
            "password": "Securepass123#@",
            "full_name": "Test User 7",
            "mobile": "0909876543",
            "address": "987 Custom Street, Hanoi"
        },
        # Test case 8: Email đã tồn tại (sẽ fail nếu test case 1 đã chạy thành công)
        {
            "email": "bob@gmail.com",
            "password": "123456",
            "full_name": "Test User 8 Duplicate",
            "mobile": "0901111111",
            "address": "Duplicate Address"
        },
        # Test case 9: User hợp lệ với full_name trống (có thể hợp lệ)
        {
            "email": "test9@example.com",
            "password": "Password999#@",
            "full_name": "Nguyễn Dũng",
            "mobile": "0909999999",
            "address": "999 Test Street"
        },
        # Test case 10: User hợp lệ với password đúng độ dài tối thiểu
        {
            "email": "test10@example.com",
            "password": "Ab234@",
            "full_name": "Test User 10",
            "mobile": "0901010101",
            "address": "1010 Test Avenue"
        }
    ]
    
    print_section("Bắt đầu script test đăng ký user")
    info(f"Tổng số test cases: {len(test_users)}")
    print()
    
    # Thống kê kết quả
    success_count = 0
    error_count = 0
    login_success_count = 0
    login_fail_count = 0
    
    # Mảng lưu ID của các user đăng ký thành công
    registered_user_ids: List[str] = []
    
    # Quét từng bản ghi
    for idx, user_data in enumerate(test_users, 1):
        print("=" * 80)
        info(f"[{idx}/{len(test_users)}] Test Case {idx}")
        print("=" * 80)
        print(f"Email: {user_data.get('email', 'N/A')}")
        print(f"Full Name: {user_data.get('full_name', 'N/A')}")
        print(f"Mobile: {user_data.get('mobile', 'N/A')}")
        print(f"Address: {user_data.get('address', 'N/A')}")
        print(f"Password: {user_data.get('password', 'N/A')}")
        print()
        
        # Đăng ký user
        register_success, user_info, error_msg = register_user(user_data)
        
        if register_success:
            success_count += 1
            
            # Lưu user ID vào mảng nếu đăng ký thành công
            if user_info and user_info.get('id'):
                user_id = user_info.get('id')
                registered_user_ids.append(user_id)
            
            # Hiển thị thông tin user đã đăng ký
            if user_info:
                print(f"   User ID: {user_info.get('id', 'N/A')}")
                print(f"   Email: {user_info.get('email', 'N/A')}")
                print(f"   Full Name: {user_info.get('full_name', 'N/A')}")
                print(f"   Mobile: {user_info.get('mobile', 'N/A')}")
                print(f"   Address: {user_info.get('address', 'N/A')}")
            
            print()
            
            # Test login sau khi đăng ký thành công
            login_success, token, login_error = login_safe(
                user_data.get("email", ""),
                user_data.get("password", "")
            )
            
            if login_success:
                login_success_count += 1
            else:
                login_fail_count += 1
                error(f"❌ Login thất bại: {login_error}")
        else:
            error_count += 1
            error(f"Đăng ký thất bại: {error_msg}")
        
        print()
    
    # Báo cáo kết quả tổng hợp
    print()
    print_section("KẾT QUẢ TỔNG HỢP")
    
    print(f"📊 Tổng số test cases: {len(test_users)}")
    print()
    
    print("📝 Kết quả đăng ký:")
    print(f"   ✅ Thành công: {success_count}")
    print(f"   ❌ Thất bại: {error_count}")
    print()
    
    print("🔐 Kết quả login sau khi đăng ký:")
    print(f"   ✅ Thành công: {login_success_count}")
    print(f"   ❌ Thất bại: {login_fail_count}")
    print()
    
    if success_count > 0:
        success(f"Tổng cộng có {success_count} user đã được đăng ký thành công!")
    
    if error_count > 0:
        error(f"Tổng cộng có {error_count} user đăng ký thất bại (có thể do validation hoặc email trùng).")
    
    print()
    
    # Hiển thị danh sách và xóa user đã đăng ký thành công bằng super_admin
    if registered_user_ids:
        print()
        print("=" * 80)
        print("=" * 80)
        
        # Login với super_admin để lấy thông tin user
        login_success, super_admin_token, login_error = login_account("super_admin")
        
        if not login_success:
            error(f"Không thể đăng nhập với super_admin: {login_error}")
            print("Không thể hiển thị danh sách user và xóa users.")
        else:
            # Hiển thị danh sách user đã đăng ký
            display_registered_users(super_admin_token, registered_user_ids)
            
            # Đợi người dùng xác nhận trước khi xóa
            if confirm_reset("xóa tất cả các user đã đăng ký ở trên"):
                print()
                print_section("BẮT ĐẦU XÓA CÁC USER ĐÃ ĐĂNG KÝ")
                info(f"Tổng số user sẽ bị xóa: {len(registered_user_ids)}")
                print()
                
                # Xóa từng user
                delete_success_count = 0
                delete_fail_count = 0
                
                for user_id in registered_user_ids:
                    delete_success, delete_error = delete_user(super_admin_token, user_id)
                    if delete_success:
                        delete_success_count += 1
                    else:
                        delete_fail_count += 1
                        error(f"Xóa user ID {user_id} thất bại: {delete_error}")
                    print()
                
                # Báo cáo kết quả xóa
                print()
                print_section("KẾT QUẢ XÓA USERS")
                print(f"   ✅ Xóa thành công: {delete_success_count}")
                print(f"   ❌ Xóa thất bại: {delete_fail_count}")
                print()
            else:
                print()
                info("Đã hủy việc xóa users. Các user đã đăng ký vẫn còn trong hệ thống.")
                print()
    else:
        info("Không có user nào được đăng ký thành công để xóa.")
        print()

if __name__ == "__main__":
    main()

