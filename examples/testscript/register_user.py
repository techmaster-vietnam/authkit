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

from share import info, success, error, get_base_url

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
            # Lấy thông báo lỗi từ response
            # goerrorkit có thể trả về error ở nhiều format khác nhau
            error_msg = "Lỗi không xác định"
            error_details = {}
            
            # Thử lấy từ "error" object (nếu là dict)
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
                error_details = error_obj.get("data", {})
            elif isinstance(error_obj, str):
                # Nếu error là string
                error_msg = error_obj
            
            # Thử lấy từ top level "message" (format của goerrorkit)
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            
            # Thử lấy từ top level "data" (chi tiết validation)
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

def test_login(email: str, password: str) -> Tuple[bool, Optional[str], Optional[str]]:
    """
    Test login với email và password
    
    Args:
        email: Email để login
        password: Password để login
    
    Returns:
        Tuple (success, token, error_message)
        - success: True nếu login thành công, False nếu lỗi
        - token: JWT token nếu thành công, None nếu lỗi
        - error_message: Thông báo lỗi nếu có, None nếu thành công
    """
    base_url = get_base_url()
    
    try:
        info(f"Đang test login với email: {email}...")
        resp = requests.post(
            f"{base_url}/api/auth/login",
            json={"email": email, "password": password},
            timeout=10
        )
        
        # Parse response
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, None, f"Response không phải JSON. Status: {resp.status_code}"
        
        # Kiểm tra lỗi
        if resp.status_code != 200:
            # Lấy thông báo lỗi từ response (tương tự như register_user)
            error_msg = "Lỗi đăng nhập không xác định"
            
            # Thử lấy từ "error" object (nếu là dict)
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
            elif isinstance(error_obj, str):
                # Nếu error là string
                error_msg = error_obj
            
            # Thử lấy từ top level "message" (format của goerrorkit)
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            
            return False, None, error_msg
        
        # Lấy token
        token = resp_data.get("data", {}).get("token")
        if not token:
            return False, None, "Không tìm thấy token trong response"
        
        success(f"Login thành công! Token: {token[:50]}...")
        return True, token, None
        
    except requests.exceptions.RequestException as e:
        return False, None, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, None, f"Lỗi không xác định: {str(e)}"

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
    
    print()
    print("=" * 80)
    info("Bắt đầu script test đăng ký user")
    print("=" * 80)
    print()
    info(f"Tổng số test cases: {len(test_users)}")
    print()
    
    # Thống kê kết quả
    success_count = 0
    error_count = 0
    login_success_count = 0
    login_fail_count = 0
    
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
            
            # Hiển thị thông tin user đã đăng ký
            if user_info:
                print(f"   User ID: {user_info.get('id', 'N/A')}")
                print(f"   Email: {user_info.get('email', 'N/A')}")
                print(f"   Full Name: {user_info.get('full_name', 'N/A')}")
                print(f"   Mobile: {user_info.get('mobile', 'N/A')}")
                print(f"   Address: {user_info.get('address', 'N/A')}")
            
            print()
            
            # Test login sau khi đăng ký thành công
            login_success, token, login_error = test_login(
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
    print("=" * 80)
    print("=" * 80)
    info("KẾT QUẢ TỔNG HỢP")
    print("=" * 80)
    print("=" * 80)
    print()
    
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
    
    print("=" * 80)
    print("=" * 80)

if __name__ == "__main__":
    main()

