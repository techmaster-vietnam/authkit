#!/usr/bin/env python3
"""Script test đổi mật khẩu:
- POST /api/auth/change-password - Đổi mật khẩu của user đã đăng nhập

Test flow:
1. Login với bob@gmail.com, password: 123456
2. POST /api/auth/change-password với old_password: 123456, new_password: 12345678
3. Logout
4. Login với bob@gmail.com, password: 123456 (sẽ fail vì đã đổi mật khẩu)
5. Login với bob@gmail.com, password: 12345678 (sẽ thành công)
6. POST /api/auth/change-password với old_password: 12345678, new_password: 123456 (đổi lại về mật khẩu cũ)
"""
import json
import sys
from typing import Tuple, Optional

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

from share import (
    info, success, error, get_base_url, handle_error_response,
    print_section, login_safe
)


def logout(token: str) -> Tuple[bool, Optional[str]]:
    """
    Đăng xuất user
    
    Args:
        token: JWT token để xác thực
    
    Returns:
        Tuple (success, error_message)
    """
    base_url = get_base_url()
    
    try:
        info("Đang đăng xuất...")
        resp = requests.post(
            f"{base_url}/api/auth/logout",
            headers={"Authorization": f"Bearer {token}"},
            timeout=10
        )
        
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, f"Response không phải JSON. Status: {resp.status_code}"
        
        if resp.status_code != 200:
            error_msg = "Lỗi logout không xác định"
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
            elif isinstance(error_obj, str):
                error_msg = error_obj
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            return False, error_msg
        
        success("Đăng xuất thành công!")
        return True, None
        
    except requests.exceptions.RequestException as e:
        return False, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, f"Lỗi không xác định: {str(e)}"


def change_password(token: str, old_password: str, new_password: str) -> Tuple[bool, Optional[str]]:
    """
    Đổi mật khẩu của user đã đăng nhập
    
    Args:
        token: JWT token để xác thực
        old_password: Mật khẩu cũ
        new_password: Mật khẩu mới
    
    Returns:
        Tuple (success, error_message)
    """
    base_url = get_base_url()
    
    try:
        info(f"Đang đổi mật khẩu...")
        info(f"  - Mật khẩu hiện tại: {'*' * len(old_password)} ({len(old_password)} ký tự)")
        info(f"  - Mật khẩu mới: {'*' * len(new_password)} ({len(new_password)} ký tự)")
        
        resp = requests.post(
            f"{base_url}/api/auth/change-password",
            json={
                "old_password": old_password,
                "new_password": new_password
            },
            headers={"Authorization": f"Bearer {token}"},
            timeout=10
        )
        
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, f"Response không phải JSON. Status: {resp.status_code}"
        
        if resp.status_code != 200:
            error_msg = "Lỗi đổi mật khẩu không xác định"
            
            # Thử lấy từ "error" object (nếu là dict)
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
            elif isinstance(error_obj, str):
                error_msg = error_obj
            
            # Thử lấy từ top level "message" (format của goerrorkit)
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            
            # Hiển thị chi tiết lỗi
            handle_error_response(resp_data, "đổi mật khẩu")
            return False, error_msg
        
        success("Đổi mật khẩu thành công!")
        return True, None
        
    except requests.exceptions.RequestException as e:
        return False, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, f"Lỗi không xác định: {str(e)}"


def main():
    """Hàm main để test đổi mật khẩu"""
    
    print_section("🧪 TEST ĐỔI MẬT KHẨU")
    
    email = "bob@gmail.com"
    old_password = "123456"
    new_password = "12345678"
    
    info(f"Email test: {email}")
    info(f"Old password: {old_password}")
    info(f"New password: {new_password}")
    info(f"Base URL: {get_base_url()}")
    print()
    
    # ========== TEST 1: LOGIN VỚI MẬT KHẨU CŨ ==========
    print_section("TEST 1: Đăng nhập với mật khẩu cũ")
    
    login_success, token, error_msg = login_safe(email, old_password)
    
    if not login_success:
        error(f"❌ Không thể đăng nhập với mật khẩu cũ: {error_msg}")
        sys.exit(1)
    
    info(f"Token: {token[:50]}...")
    print()
    
    # ========== TEST 2: ĐỔI MẬT KHẨU ==========
    print_section("TEST 2: Đổi mật khẩu")
    
    change_success, change_error = change_password(token, old_password, new_password)
    
    if not change_success:
        error(f"❌ Đổi mật khẩu thất bại: {change_error}")
        sys.exit(1)
    
    print()
    
    # ========== TEST 3: LOGOUT ==========
    print_section("TEST 3: Đăng xuất")
    
    logout_success, logout_error = logout(token)
    
    if not logout_success:
        error(f"⚠️  Logout thất bại: {logout_error}")
        info("   Tiếp tục test...")
    
    print()
    
    # ========== TEST 4: LOGIN VỚI MẬT KHẨU CŨ (SẼ FAIL) ==========
    print_section("TEST 4: Đăng nhập với mật khẩu cũ (nên thất bại)")
    
    login_old_success, token_old, error_msg_old = login_safe(email, old_password)
    
    if login_old_success:
        error("⚠️  Đăng nhập với mật khẩu cũ vẫn thành công (không mong đợi)")
        error("   Mật khẩu có thể chưa được đổi hoặc có vấn đề")
    else:
        success("✅ Đăng nhập với mật khẩu cũ thất bại (đúng như mong đợi)")
        info(f"   Lỗi: {error_msg_old}")
    
    print()
    
    # ========== TEST 5: LOGIN VỚI MẬT KHẨU MỚI ==========
    print_section("TEST 5: Đăng nhập với mật khẩu mới")
    
    login_new_success, token_new, error_msg_new = login_safe(email, new_password)
    
    if not login_new_success:
        error(f"❌ Không thể đăng nhập với mật khẩu mới: {error_msg_new}")
        error("   Có thể mật khẩu chưa được đổi thành công")
        sys.exit(1)
    
    info(f"Token: {token_new[:50]}...")
    print()
    
    # ========== TEST 6: ĐỔI LẠI VỀ MẬT KHẨU CŨ ==========
    print_section("TEST 6: Đổi lại về mật khẩu cũ")
    
    change_back_success, change_back_error = change_password(token_new, new_password, old_password)
    
    if not change_back_success:
        error(f"❌ Đổi lại mật khẩu cũ thất bại: {change_back_error}")
        sys.exit(1)
    
    print()
    
    # ========== TEST 7: XÁC MINH MẬT KHẨU ĐÃ ĐƯỢC ĐỔI LẠI ==========
    print_section("TEST 7: Xác minh mật khẩu đã được đổi lại")
    
    # Logout trước
    logout(token_new)
    print()
    
    # Login với mật khẩu cũ (nên thành công)
    login_final_success, token_final, error_msg_final = login_safe(email, old_password)
    
    if login_final_success:
        success("✅ Đăng nhập với mật khẩu cũ thành công (đã đổi lại)")
        info(f"Token: {token_final[:50]}...")
    else:
        error(f"❌ Không thể đăng nhập với mật khẩu cũ sau khi đổi lại: {error_msg_final}")
        sys.exit(1)
    
    print()
    
    # ========== TỔNG KẾT ==========
    print_section("📊 TỔNG KẾT")
    
    results = {
        "Login với mật khẩu cũ (lần 1)": "✅" if login_success else "❌",
        "Đổi mật khẩu": "✅" if change_success else "❌",
        "Logout": "✅" if logout_success else "⚠️",
        "Login với mật khẩu cũ (sau khi đổi)": "✅ (đúng như mong đợi - fail)" if not login_old_success else "❌ (không mong đợi)",
        "Login với mật khẩu mới": "✅" if login_new_success else "❌",
        "Đổi lại về mật khẩu cũ": "✅" if change_back_success else "❌",
        "Xác minh mật khẩu đã đổi lại": "✅" if login_final_success else "❌",
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


if __name__ == "__main__":
    main()

