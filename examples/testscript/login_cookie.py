#!/usr/bin/env python3
"""Script test login, refresh và logout với cookie support
Test các endpoint:
- POST /api/auth/login - Đăng nhập, nhận access token và refresh token trong cookie
- POST /api/auth/refresh - Làm mới access token bằng refresh token từ cookie
- POST /api/auth/logout - Đăng xuất, xóa refresh token và cookie
"""
import json
import sys
import time
from typing import Dict, Optional, Tuple

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

from share import info, success, error, get_base_url, handle_error_response

# Colors cho print_section (chỉ cần BLUE)
BLUE = '\033[0;34m'
RESET = '\033[0m'


def print_section(title: str):
    """In tiêu đề section"""
    print()
    print("=" * 80)
    print(f"{BLUE}{title}{RESET}")
    print("=" * 80)
    print()


def parse_response(resp: requests.Response, operation: str = "thao tác") -> Tuple[Optional[Dict], bool]:
    """
    Parse response và xử lý lỗi chung
    
    Args:
        resp: Response object từ requests
        operation: Tên thao tác đang thực hiện (để hiển thị trong thông báo lỗi)
    
    Returns:
        Tuple (resp_data, success) - resp_data là None nếu có lỗi
    """
    try:
        resp_data = resp.json()
    except json.JSONDecodeError:
        error(f"Response không phải JSON. Status: {resp.status_code}")
        error(f"Response body: {resp.text}")
        return None, False
    
    # Kiểm tra lỗi
    if resp.status_code != 200:
        handle_error_response(resp_data, operation)
        return None, False
    
    return resp_data, True


def test_login(session: requests.Session, email: str, password: str) -> Tuple[bool, Optional[str], Optional[Dict]]:
    """
    Test đăng nhập và lưu refresh token vào cookie
    
    Args:
        session: requests.Session để quản lý cookie tự động
        email: Email để đăng nhập
        password: Password để đăng nhập
    
    Returns:
        Tuple (success, access_token, user_info)
    """
    base_url = get_base_url()
    
    info(f"🔐 Đang đăng nhập với email: {email}...")
    
    try:
        resp = session.post(
            f"{base_url}/api/auth/login",
            json={"email": email, "password": password},
            timeout=10
        )
        
        # Parse response và xử lý lỗi
        resp_data, success_parse = parse_response(resp, "đăng nhập")
        if not success_parse:
            return False, None, None
        
        # Kiểm tra response có data không
        if "data" not in resp_data:
            error("Response không chứa data")
            return False, None, None
        
        # Lấy access token và user info
        data = resp_data.get("data", {})
        access_token = data.get("token")
        user_info = data.get("user", {})
        
        if not access_token:
            error("Không tìm thấy access token trong response")
            return False, None, None
        
        # Kiểm tra cookie refresh_token
        refresh_token_cookie = session.cookies.get("refresh_token")
        if refresh_token_cookie:
            success("✅ Đăng nhập thành công!")
            info(f"   Access Token: {access_token[:50]}...")
            info(f"   Refresh Token Cookie: {refresh_token_cookie[:30]}... (đã được lưu tự động)")
            info(f"   User ID: {user_info.get('id', 'N/A')}")
            info(f"   Email: {user_info.get('email', 'N/A')}")
            info(f"   Full Name: {user_info.get('full_name', 'N/A')}")
        else:
            error("⚠️  Đăng nhập thành công nhưng không có refresh token cookie!")
            error("   Lưu ý: Cookie có thể không được set nếu Secure=true và đang test trên HTTP")
            return False, None, None
        
        return True, access_token, user_info
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi kết nối: {str(e)}")
        return False, None, None
    except Exception as e:
        error(f"Lỗi không xác định: {str(e)}")
        return False, None, None


def test_refresh(session: requests.Session) -> Tuple[bool, Optional[str]]:
    """
    Test làm mới access token bằng refresh token từ cookie
    
    Args:
        session: requests.Session đã có cookie refresh_token
    
    Returns:
        Tuple (success, new_access_token)
    """
    base_url = get_base_url()
    
    info("🔄 Đang làm mới access token...")
    
    # Kiểm tra có cookie refresh_token không
    refresh_token_cookie = session.cookies.get("refresh_token")
    if not refresh_token_cookie:
        error("Không có refresh token cookie trong session!")
        error("   Hãy đăng nhập trước khi refresh token")
        return False, None
    
    info(f"   Refresh Token Cookie: {refresh_token_cookie[:30]}...")
    
    try:
        resp = session.post(
            f"{base_url}/api/auth/refresh",
            timeout=10
        )
        
        # Parse response và xử lý lỗi
        resp_data, success_parse = parse_response(resp, "refresh token")
        if not success_parse:
            # Kiểm tra cookie có bị xóa không (nếu token không hợp lệ)
            new_refresh_token = session.cookies.get("refresh_token")
            if not new_refresh_token:
                info("   Cookie refresh_token đã bị xóa (token không hợp lệ)")
            return False, None
        
        # Kiểm tra response có data không
        if "data" not in resp_data:
            error("Response không chứa data")
            return False, None
        
        # Lấy access token mới
        data = resp_data.get("data", {})
        new_access_token = data.get("token")
        
        if not new_access_token:
            error("Không tìm thấy access token mới trong response")
            return False, None
        
        # Kiểm tra refresh token mới trong cookie (rotation)
        new_refresh_token = session.cookies.get("refresh_token")
        if new_refresh_token and new_refresh_token != refresh_token_cookie:
            success("✅ Refresh token thành công!")
            info(f"   Access Token mới: {new_access_token[:50]}...")
            info(f"   Refresh Token Cookie mới: {new_refresh_token[:30]}... (đã được rotate)")
        else:
            success("✅ Refresh token thành công!")
            info(f"   Access Token mới: {new_access_token[:50]}...")
            if new_refresh_token == refresh_token_cookie:
                info("   ⚠️  Refresh token cookie không thay đổi (có thể không có rotation)")
        
        return True, new_access_token
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi kết nối: {str(e)}")
        return False, None
    except Exception as e:
        error(f"Lỗi không xác định: {str(e)}")
        return False, None


def test_logout(session: requests.Session) -> bool:
    """
    Test đăng xuất, xóa refresh token và cookie
    
    Args:
        session: requests.Session đã có cookie refresh_token
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    base_url = get_base_url()
    
    info("🚪 Đang đăng xuất...")
    
    # Kiểm tra có cookie refresh_token không
    refresh_token_cookie = session.cookies.get("refresh_token")
    if not refresh_token_cookie:
        info("   Không có refresh token cookie trong session")
        info("   Vẫn sẽ gửi request logout để xóa cookie nếu có")
    
    try:
        resp = session.post(
            f"{base_url}/api/auth/logout",
            timeout=10
        )
        
        # Parse response và xử lý lỗi
        resp_data, success_parse = parse_response(resp, "logout")
        if not success_parse:
            return False
        
        # Kiểm tra cookie đã bị xóa chưa
        remaining_refresh_token = session.cookies.get("refresh_token")
        if not remaining_refresh_token:
            success("✅ Đăng xuất thành công!")
            info("   Refresh token cookie đã được xóa")
        else:
            success("✅ Đăng xuất thành công!")
            info(f"   ⚠️  Cookie refresh_token vẫn còn: {remaining_refresh_token[:30]}...")
            info("   (Có thể do cookie Secure=true trên HTTP localhost)")
        
        return True
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi kết nối: {str(e)}")
        return False
    except Exception as e:
        error(f"Lỗi không xác định: {str(e)}")
        return False


def test_access_token_expiry(session: requests.Session, access_token: str, wait_seconds: int = 35):
    """
    Test xem access token có hết hạn sau khi chờ không
    
    Args:
        session: requests.Session
        access_token: Access token để test
        wait_seconds: Số giây chờ (mặc định 35 giây để test với JWT_EXPIRATION_HOURS=0.00833 ~ 30 giây)
    """
    base_url = get_base_url()
    
    print_section("⏱️  Test Access Token Expiry")
    
    info(f"Đang chờ {wait_seconds} giây để test access token expiry...")
    info("   (JWT_EXPIRATION_HOURS trong .env nên được set thành 0.00833 ~ 30 giây để test)")
    
    for i in range(wait_seconds, 0, -10):
        print(f"   Còn lại: {i} giây...", end='\r')
        time.sleep(min(10, i))
    
    print("   Còn lại: 0 giây...")
    print()
    
    info("Đang test gọi API với access token đã hết hạn...")
    
    try:
        resp = session.get(
            f"{base_url}/api/auth/profile",
            headers={"Authorization": f"Bearer {access_token}"},
            timeout=10
        )
        
        if resp.status_code == 401:
            success("✅ Access token đã hết hạn (401 Unauthorized)")
            info("   Đây là hành vi mong đợi!")
        elif resp.status_code == 200:
            error("⚠️  Access token vẫn còn hiệu lực sau khi chờ")
            error("   Có thể JWT_EXPIRATION_HOURS chưa được set đúng trong .env")
        else:
            info(f"   Status code: {resp.status_code}")
            
    except requests.exceptions.RequestException as e:
        error(f"Lỗi kết nối: {str(e)}")


def main():
    """Hàm main để test login, refresh và logout với cookie"""
    
    print_section("🧪 TEST LOGIN, REFRESH VÀ LOGOUT VỚI COOKIE")
    
    # Thông tin đăng nhập
    email = "bob@gmail.com"
    password = "123456"
    
    info(f"Email test: {email}")
    info(f"Base URL: {get_base_url()}")
    print()
    
    # Tạo session để quản lý cookie tự động
    session = requests.Session()
    
    # ========== TEST 1: LOGIN ==========
    print_section("TEST 1: Đăng nhập (Login)")
    
    login_success, access_token, user_info = test_login(session, email, password)
    
    if not login_success:
        error("❌ Test login thất bại, không thể tiếp tục test refresh và logout")
        sys.exit(1)
    
    # Lưu access token để test sau
    original_access_token = access_token
    
    # ========== TEST 2: REFRESH TOKEN ==========
    print_section("TEST 2: Làm mới access token (Refresh)")
    
    refresh_success, new_access_token = test_refresh(session)
    
    if not refresh_success:
        error("❌ Test refresh token thất bại")
    else:
        # Cập nhật access token mới
        access_token = new_access_token
    
    # ========== TEST 3: REFRESH TOKEN LẦN 2 (Test Rotation) ==========
    print_section("TEST 3: Refresh token lần 2 (Test Token Rotation)")
    
    refresh_success2, new_access_token2 = test_refresh(session)
    
    if not refresh_success2:
        error("❌ Test refresh token lần 2 thất bại")
    else:
        # Kiểm tra token có khác nhau không
        if new_access_token2 != new_access_token:
            success("✅ Access token đã được rotate (token mới khác token cũ)")
        else:
            info("⚠️  Access token không thay đổi sau refresh lần 2")
    
    # ========== TEST 4: LOGOUT ==========
    print_section("TEST 4: Đăng xuất (Logout)")
    
    logout_success = test_logout(session)
    
    if not logout_success:
        error("❌ Test logout thất bại")
    
    # ========== TEST 5: REFRESH SAU KHI LOGOUT ==========
    print_section("TEST 5: Refresh token sau khi logout (Nên thất bại)")
    
    refresh_after_logout_success, _ = test_refresh(session)
    
    if refresh_after_logout_success:
        error("⚠️  Refresh token vẫn hoạt động sau khi logout (không mong đợi)")
    else:
        success("✅ Refresh token đã bị vô hiệu hóa sau khi logout (đúng như mong đợi)")
    
    # ========== TEST 6: LOGIN LẠI ==========
    print_section("TEST 6: Đăng nhập lại sau khi logout")
    
    login_success2, access_token2, user_info2 = test_login(session, email, password)
    
    if not login_success2:
        error("❌ Test login lại thất bại")
    
    # ========== TEST 7: ACCESS TOKEN EXPIRY (Optional) ==========
    # Chỉ test nếu JWT_EXPIRATION_HOURS được set nhỏ trong .env
    print_section("TEST 7: Test Access Token Expiry (Optional)")
    
    info("⚠️  Test này sẽ chờ ~35 giây để kiểm tra access token expiry")
    info("   (JWT_EXPIRATION_HOURS trong .env nên được set thành 0.00833 ~ 30 giây)")
    info("   Để skip test này, hãy comment out phần code này")
    print()
    
    user_input = input("Bạn có muốn test access token expiry? (y/n, mặc định: n): ").strip().lower()
    
    if user_input == 'y':
        test_access_token_expiry(session, access_token2, wait_seconds=35)
    else:
        info("⏭️  Đã skip test access token expiry")
    
    # ========== TỔNG KẾT ==========
    print_section("📊 TỔNG KẾT")
    
    results = {
        "Login": "✅" if login_success else "❌",
        "Refresh": "✅" if refresh_success else "❌",
        "Refresh lần 2": "✅" if refresh_success2 else "❌",
        "Logout": "✅" if logout_success else "❌",
        "Refresh sau logout": "✅ (đúng như mong đợi)" if not refresh_after_logout_success else "❌ (không mong đợi)",
        "Login lại": "✅" if login_success2 else "❌",
    }
    
    for test_name, result in results.items():
        print(f"   {test_name}: {result}")
    
    print()
    
    # Đếm số test thành công
    success_count = sum(1 for v in results.values() if "✅" in v)
    total_count = len(results)
    
    if success_count == total_count:
        success(f"🎉 Tất cả {total_count} tests đều thành công!")
    else:
        error(f"⚠️  {success_count}/{total_count} tests thành công")
    
    print()
    print("=" * 80)
    print()
    
    # Lưu ý về Secure cookie
    info("📝 LƯU Ý:")
    info("   - Cookie refresh_token có Secure=true trong code Go")
    info("   - Khi test trên localhost HTTP, cookie có thể không được set/gửi")
    info("   - Để test đầy đủ, có thể cần:")
    info("     1. Set Secure=false trong handlers/base_auth_handler.go khi test")
    info("     2. Hoặc test trên HTTPS (localhost với self-signed cert)")
    print()


if __name__ == "__main__":
    main()

