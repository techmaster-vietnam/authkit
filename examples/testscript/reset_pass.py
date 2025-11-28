#!/usr/bin/env python3
"""Script test password reset và change password flow

Kịch bản:
1. Anonymous user POST /auth/request-password-reset với email bob@gmail.com
2. Go app tạo reset token rồi lưu vào một json
3. Đọc reset token từ file json rồi POST /auth/reset-password với password mới 12345678
4. Login với bob@gmail.com, pass:12345678
5. Change-password từ 12345678 về 123456
6. Login với bob@gmail.com, pass:123456
"""
import json
import os
import sys
import time
from typing import Optional, Tuple

# Thêm thư mục cha vào path để import share
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from share import (
    get_base_url,
    info,
    success,
    error,
    login_safe,
    print_section,
    handle_error_response,
)

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

# Đường dẫn file chứa reset tokens
RESET_TOKENS_FILE = os.path.join(os.path.dirname(__file__), "reset_tokens.json")


def read_reset_token(email: str) -> Optional[str]:
    """Đọc reset token từ file JSON"""
    if not os.path.exists(RESET_TOKENS_FILE):
        error(f"File {RESET_TOKENS_FILE} không tồn tại")
        return None

    try:
        with open(RESET_TOKENS_FILE, "r", encoding="utf-8") as f:
            tokens = json.load(f)
        
        if email not in tokens:
            error(f"Không tìm thấy reset token cho email: {email}")
            return None
        
        token_data = tokens[email]
        if isinstance(token_data, dict):
            return token_data.get("token")
        # Nếu token_data là string (format cũ)
        return token_data
    except json.JSONDecodeError as e:
        error(f"Lỗi khi parse JSON: {e}")
        return None
    except Exception as e:
        error(f"Lỗi khi đọc file: {e}")
        return None


def request_password_reset(email: str) -> bool:
    """Gửi yêu cầu reset password (anonymous user)"""
    info(f"Gửi yêu cầu reset password cho email: {email}")
    
    url = f"{get_base_url()}/api/auth/request-password-reset"
    data = {"email": email}
    
    try:
        # Anonymous request - không cần Authorization header
        response = requests.post(url, json=data, timeout=10)
        
        if response.status_code == 200:
            success("Yêu cầu reset password đã được gửi thành công")
            try:
                resp_data = response.json()
                import json
                print(json.dumps(resp_data, indent=2, ensure_ascii=False))
            except:
                print(f"Response: {response.text}")
            
            # Đợi một chút để đảm bảo file đã được ghi
            time.sleep(0.5)
            
            # Đọc token từ file
            token = read_reset_token(email)
            if token:
                success(f"Reset token đã được lưu: {token[:20]}...")
                return True
            else:
                error("Không thể đọc reset token từ file")
                return False
        else:
            error(f"Yêu cầu reset password thất bại: {response.status_code}")
            try:
                resp_data = response.json()
                handle_error_response(resp_data, "yêu cầu reset password")
            except:
                print(f"Response: {response.text}")
            return False
    except requests.exceptions.RequestException as e:
        error(f"Lỗi kết nối: {str(e)}")
        return False
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return False


def reset_password_with_token(token: str, new_password: str) -> bool:
    """Đặt lại mật khẩu bằng reset token"""
    info(f"Đặt lại mật khẩu với token: {token[:20]}...")
    
    url = f"{get_base_url()}/api/auth/reset-password"
    data = {
        "token": token,
        "new_password": new_password,
    }
    
    try:
        # Anonymous request - không cần Authorization header
        response = requests.post(url, json=data, timeout=10)
        
        if response.status_code == 200:
            success("Đặt lại mật khẩu thành công")
            try:
                resp_data = response.json()
                import json
                print(json.dumps(resp_data, indent=2, ensure_ascii=False))
            except:
                print(f"Response: {response.text}")
            return True
        else:
            error(f"Đặt lại mật khẩu thất bại: {response.status_code}")
            try:
                resp_data = response.json()
                handle_error_response(resp_data, "đặt lại mật khẩu")
            except:
                print(f"Response: {response.text}")
            return False
    except requests.exceptions.RequestException as e:
        error(f"Lỗi kết nối: {str(e)}")
        return False
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return False


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
    """Hàm main để test password reset và change password flow"""
    
    print_section("🧪 TEST PASSWORD RESET VÀ CHANGE PASSWORD FLOW")
    
    email = "bob@gmail.com"
    reset_password_new = "12345678"
    final_password = "123456"
    
    info(f"Email test: {email}")
    info(f"Password sau reset: {reset_password_new}")
    info(f"Password sau change: {final_password}")
    info(f"Base URL: {get_base_url()}")
    print()
    
    # ========== BƯỚC 1: REQUEST PASSWORD RESET ==========
    print_section("BƯỚC 1: Anonymous user yêu cầu reset password")
    
    if not request_password_reset(email):
        error("❌ Không thể gửi yêu cầu reset password")
        sys.exit(1)
    
    # Đọc token từ file
    reset_token = read_reset_token(email)
    if not reset_token:
        error("❌ Không thể đọc reset token từ file")
        sys.exit(1)
    
    info(f"Reset token đã được đọc: {reset_token[:20]}...")
    print()
    
    # ========== BƯỚC 2: RESET PASSWORD VỚI TOKEN ==========
    print_section("BƯỚC 2: Đặt lại mật khẩu bằng reset token")
    
    if not reset_password_with_token(reset_token, reset_password_new):
        error("❌ Không thể đặt lại mật khẩu")
        sys.exit(1)
    
    print()
    
    # ========== BƯỚC 3: LOGIN VỚI MẬT KHẨU MỚI ==========
    print_section("BƯỚC 3: Đăng nhập với mật khẩu mới sau reset")
    
    login_success, login_token, login_error = login_safe(email, reset_password_new)
    
    if not login_success:
        error(f"❌ Không thể đăng nhập với mật khẩu mới: {login_error}")
        sys.exit(1)
    
    success("Đăng nhập thành công với mật khẩu mới!")
    info(f"Token: {login_token[:50]}...")
    print()
    
    # ========== BƯỚC 4: CHANGE PASSWORD ==========
    print_section("BƯỚC 4: Đổi mật khẩu từ reset password về password cũ")
    
    change_success, change_error = change_password(login_token, reset_password_new, final_password)
    
    if not change_success:
        error(f"❌ Đổi mật khẩu thất bại: {change_error}")
        sys.exit(1)
    
    print()
    
    # ========== BƯỚC 5: LOGIN VỚI MẬT KHẨU SAU KHI CHANGE ==========
    print_section("BƯỚC 5: Đăng nhập với mật khẩu sau khi change")
    
    login_final_success, login_final_token, login_final_error = login_safe(email, final_password)
    
    if not login_final_success:
        error(f"❌ Không thể đăng nhập với mật khẩu sau khi change: {login_final_error}")
        sys.exit(1)
    
    success("Đăng nhập thành công với mật khẩu sau khi change!")
    info(f"Token: {login_final_token[:50]}...")
    print()
    
    # ========== TỔNG KẾT ==========
    print_section("📊 TỔNG KẾT")
    
    results = {
        "Request password reset": "✅",
        "Đọc reset token từ file": "✅" if reset_token else "❌",
        "Reset password với token": "✅" if change_success else "❌",
        "Login với mật khẩu mới (12345678)": "✅" if login_success else "❌",
        "Change password về 123456": "✅" if change_success else "❌",
        "Login với mật khẩu cuối (123456)": "✅" if login_final_success else "❌",
    }
    
    for step_name, result in results.items():
        print(f"   {step_name}: {result}")
    
    print()
    
    # Đếm số test thành công
    success_count = sum(1 for v in results.values() if "✅" in v)
    total_count = len(results)
    
    if success_count == total_count:
        success(f"🎉 Tất cả {total_count} bước đều thành công!")
    else:
        error(f"⚠️  {success_count}/{total_count} bước thành công")
    
    print()
    print("=" * 80)
    print()


if __name__ == "__main__":
    main()

