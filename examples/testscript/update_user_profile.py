#!/usr/bin/env python3
"""Script test cập nhật user profile
Test các endpoint:
- POST /api/auth/register - Đăng ký user
- POST /api/auth/login - Đăng nhập
- PUT /api/auth/profile - Cập nhật profile của chính mình
- POST /api/auth/logout - Đăng xuất
- GET /api/auth/profile/:id - Lấy thông tin profile theo identifier (admin/super_admin)
- PUT /api/auth/profile/:id - Cập nhật profile theo ID (admin/super_admin)
- PUT /api/users/:userId/roles - Cập nhật roles cho user
- DELETE /api/auth/profile/:id - Xóa user (super_admin)
"""
import json
import sys
from typing import Dict, Optional, Tuple
from urllib.parse import quote

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

from share import (
    info, success, error, get_base_url, print_section,
    login, login_safe, login_account, get_user_detail,
    update_user_roles, delete_user, handle_error_response
)

# Import hàm register_user từ register_user.py
# Hoặc định nghĩa lại ở đây để tránh import phức tạp
def register_user(user_data: Dict[str, str]) -> Tuple[bool, Optional[Dict], Optional[str]]:
    """
    Đăng ký user mới
    
    Args:
        user_data: Dictionary chứa thông tin user (email, password, full_name, mobile, address)
    
    Returns:
        Tuple (success, user_info, error_message)
    """
    base_url = get_base_url()
    
    request_body = {
        "email": user_data.get("email", ""),
        "password": user_data.get("password", ""),
        "full_name": user_data.get("full_name", ""),
    }
    
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
        
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, None, f"Response không phải JSON. Status: {resp.status_code}"
        
        if resp.status_code != 201:
            error_msg = "Lỗi không xác định"
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
            elif isinstance(error_obj, str):
                error_msg = error_obj
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            return False, None, error_msg
        
        if "data" not in resp_data:
            return False, None, "Response không chứa data"
        
        user_info = resp_data.get("data", {})
        success(f"Đăng ký thành công! User ID: {user_info.get('id', 'N/A')}")
        return True, user_info, None
        
    except requests.exceptions.RequestException as e:
        return False, None, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, None, f"Lỗi không xác định: {str(e)}"

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

def update_profile(token: str, profile_data: Dict[str, str]) -> Tuple[bool, Optional[Dict], Optional[str]]:
    """
    Cập nhật profile của chính mình
    
    Args:
        token: JWT token để xác thực
        profile_data: Dictionary chứa thông tin cần cập nhật (mobile, address, full_name)
                      Lưu ý: email và password không được cập nhật qua endpoint này
    
    Returns:
        Tuple (success, response_data, error_message)
    """
    base_url = get_base_url()
    
    try:
        info("Đang cập nhật profile của chính mình...")
        info(f"  - Mobile: {profile_data.get('mobile', 'N/A')}")
        info(f"  - Address: {profile_data.get('address', 'N/A')}")
        
        resp = requests.put(
            f"{base_url}/api/auth/profile",
            json=profile_data,
            headers={"Authorization": f"Bearer {token}"},
            timeout=10
        )
        
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, None, f"Response không phải JSON. Status: {resp.status_code}"
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "cập nhật profile")
            return False, resp_data, "Cập nhật profile thất bại"
        
        success("Cập nhật profile thành công!")
        return True, resp_data, None
        
    except requests.exceptions.RequestException as e:
        return False, None, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, None, f"Lỗi không xác định: {str(e)}"

def update_profile_by_id(token: str, user_id: str, profile_data: Dict[str, str]) -> Tuple[bool, Optional[Dict], Optional[str]]:
    """
    Cập nhật profile của user khác (chỉ admin/super_admin)
    
    Args:
        token: JWT token để xác thực (phải là admin hoặc super_admin)
        user_id: ID của user cần cập nhật
        profile_data: Dictionary chứa thông tin cần cập nhật (mobile, address, full_name)
                      Lưu ý: email và password không được cập nhật qua endpoint này
    
    Returns:
        Tuple (success, response_data, error_message)
    """
    base_url = get_base_url()
    
    try:
        info(f"Đang cập nhật profile của user ID: {user_id}...")
        info(f"  - Mobile: {profile_data.get('mobile', 'N/A')}")
        info(f"  - Address: {profile_data.get('address', 'N/A')}")
        
        # URL encode user_id để đảm bảo an toàn
        encoded_user_id = quote(str(user_id), safe='')
        resp = requests.put(
            f"{base_url}/api/auth/profile/{encoded_user_id}",
            json=profile_data,
            headers={"Authorization": f"Bearer {token}"},
            timeout=10
        )
        
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, None, f"Response không phải JSON. Status: {resp.status_code}"
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "cập nhật profile theo ID")
            return False, resp_data, "Cập nhật profile theo ID thất bại"
        
        success("Cập nhật profile theo ID thành công!")
        return True, resp_data, None
        
    except requests.exceptions.RequestException as e:
        return False, None, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, None, f"Lỗi không xác định: {str(e)}"

def main():
    """Hàm main để test cập nhật user profile"""
    
    print_section("🧪 TEST CẬP NHẬT USER PROFILE")
    
    # Thông tin user ban đầu
    initial_email = "foo@gmail.com"
    initial_password = "123456"
    initial_mobile = "0902209011"
    initial_address = "Hồ Gươm, Hoàn Kiếm"
    
    # Thông tin user sau khi update lần 1 (chỉ mobile và address)
    updated_mobile = "0902209088"
    updated_address = "Lăng Bác, Ba Đình"
    
    user_id = None
    
    # ========== BƯỚC 1: REGISTER ==========
    print_section("BƯỚC 1: Đăng ký user")
    
    user_data = {
        "email": initial_email,
        "password": initial_password,
        "full_name": "Foo User",
        "mobile": initial_mobile,
        "address": initial_address
    }
    
    register_success, user_info, error_msg = register_user(user_data)
    
    if not register_success:
        error(f"Đăng ký thất bại: {error_msg}")
        sys.exit(1)
    
    if user_info and user_info.get('id'):
        user_id = user_info.get('id')
        info(f"User ID: {user_id}")
    
    print()
    
    # ========== BƯỚC 2: LOGIN VỚI foo@gmail.com ==========
    print_section("BƯỚC 2: Đăng nhập với foo@gmail.com")
    
    login_success, token1, login_error = login_safe(initial_email, initial_password)
    
    if not login_success:
        error(f"Đăng nhập thất bại: {login_error}")
        sys.exit(1)
    
    print()
    
    # ========== BƯỚC 3: UPDATE PROFILE ==========
    print_section("BƯỚC 3: Cập nhật profile của chính mình")
    
    info("⚠️  LƯU Ý: Chỉ cập nhật mobile và address (email và password không được cập nhật qua UpdateProfile)")
    print()
    
    profile_update_data = {
        "mobile": updated_mobile,
        "address": updated_address
    }
    
    update_success, update_resp, update_error = update_profile(token1, profile_update_data)
    
    if not update_success:
        error(f"Cập nhật profile thất bại: {update_error}")
        sys.exit(1)
    
    # Kiểm tra xem mobile và address đã được cập nhật chưa
    if update_resp and "data" in update_resp:
        updated_user = update_resp["data"]
        if updated_user.get("mobile") == updated_mobile and updated_user.get("address") == updated_address:
            success("Mobile và address đã được cập nhật thành công!")
        else:
            info(f"Mobile trong response: {updated_user.get('mobile')} (mong đợi: {updated_mobile})")
            info(f"Address trong response: {updated_user.get('address')} (mong đợi: {updated_address})")
    
    print()
    
    # ========== BƯỚC 4: LOGOUT VÀ LOGIN VỚI ADMIN ==========
    print_section("BƯỚC 4: Đăng xuất và đăng nhập với admin")
    
    logout_success, logout_error = logout(token1)
    if not logout_success:
        error(f"Đăng xuất thất bại: {logout_error}")
    
    print()
    
    # Login với admin
    admin_login_success, admin_token, admin_error = login_account("admin")
    
    if not admin_login_success:
        error(f"Đăng nhập với admin thất bại: {admin_error}")
        sys.exit(1)
    
    print()
    
    # Lấy thông tin profile của foo@gmail.com
    info(f"Đang lấy thông tin profile của {initial_email}...")
    user_detail = get_user_detail(admin_token, initial_email, verbose=True)
    
    if user_detail:
        user = user_detail.get("user", {})
        if user.get('id'):
            user_id = user.get('id')  # Cập nhật user_id nếu chưa có
        info(f"Tìm thấy user với email: {user.get('email', 'N/A')}")
        info(f"Mobile: {user.get('mobile', 'N/A')} (mong đợi: {updated_mobile})")
        info(f"Address: {user.get('address', 'N/A')} (mong đợi: {updated_address})")
    else:
        error("Không thể lấy thông tin profile")
        sys.exit(1)
    
    print()
    
    # ========== BƯỚC 5: UPDATE PROFILE BY ID ==========
    print_section("BƯỚC 5: Cập nhật profile theo ID (admin)")
    
    if not user_id:
        error("Không có user_id để cập nhật")
        sys.exit(1)
    
    info("⚠️  LƯU Ý: Chỉ cập nhật mobile và address về giá trị ban đầu")
    print()
    
    profile_reset_data = {
        "mobile": initial_mobile,
        "address": initial_address
    }
    
    update_by_id_success, update_by_id_resp, update_by_id_error = update_profile_by_id(
        admin_token, user_id, profile_reset_data
    )
    
    if not update_by_id_success:
        error(f"Cập nhật profile theo ID thất bại: {update_by_id_error}")
        sys.exit(1)
    
    # Kiểm tra xem mobile và address đã được reset chưa
    if update_by_id_resp and "data" in update_by_id_resp:
        reset_user = update_by_id_resp["data"]
        if reset_user.get("mobile") == initial_mobile and reset_user.get("address") == initial_address:
            success("Mobile và address đã được reset về giá trị ban đầu!")
        else:
            info(f"Mobile trong response: {reset_user.get('mobile')} (mong đợi: {initial_mobile})")
            info(f"Address trong response: {reset_user.get('address')} (mong đợi: {initial_address})")
    
    print()
    
    # ========== BƯỚC 6: UPDATE USER ROLES ==========
    print_section("BƯỚC 6: Cập nhật roles cho user")
    
    roles_to_update = ["reader", "editor"]
    update_roles_success, update_roles_resp = update_user_roles(admin_token, user_id, roles_to_update)
    
    if not update_roles_success:
        error("Cập nhật roles thất bại")
    
    print()
    
    # ========== BƯỚC 7: LOGOUT VÀ LOGIN LẠI VỚI foo@gmail.com ==========
    print_section("BƯỚC 7: Đăng xuất và đăng nhập lại với foo@gmail.com")
    
    logout_success2, logout_error2 = logout(admin_token)
    if not logout_success2:
        error(f"Đăng xuất thất bại: {logout_error2}")
    
    print()
    
    # Login lại với email ban đầu
    login_success2, token2, login_error2 = login_safe(initial_email, initial_password)
    
    if not login_success2:
        error(f"Đăng nhập với {initial_email} thất bại: {login_error2}")
        sys.exit(1)
    
    print()
    
    # ========== BƯỚC 8: LOGOUT, LOGIN VỚI SUPER_ADMIN, XÓA USER ==========
    print_section("BƯỚC 8: Đăng xuất, đăng nhập với super_admin, xóa user")
    
    logout_success3, logout_error3 = logout(token2)
    if not logout_success3:
        error(f"Đăng xuất thất bại: {logout_error3}")
    
    print()
    
    # Login với super_admin
    super_admin_login_success, super_admin_token, super_admin_error = login_account("super_admin")
    
    if not super_admin_login_success:
        error(f"Đăng nhập với super_admin thất bại: {super_admin_error}")
        sys.exit(1)
    
    print()
    
    # Xóa user
    if user_id:
        delete_success, delete_error = delete_user(super_admin_token, user_id)
        
        if not delete_success:
            error(f"Xóa user thất bại: {delete_error}")
        else:
            success("Xóa user thành công!")
    else:
        error("Không có user_id để xóa")
    
    print()
    
    # ========== TỔNG KẾT ==========
    print_section("📊 TỔNG KẾT")
    
    results = {
        "Đăng ký user": "✅" if register_success else "❌",
        "Đăng nhập lần 1": "✅" if login_success else "❌",
        "Cập nhật profile (mobile, address)": "✅" if update_success else "❌",
        "Lấy profile (admin)": "✅" if user_detail else "❌",
        "Cập nhật profile theo ID": "✅" if update_by_id_success else "❌",
        "Cập nhật roles": "✅" if update_roles_success else "❌",
        "Đăng nhập lại": "✅" if login_success2 else "❌",
        "Xóa user": "✅" if (user_id and delete_success) else "❌",
    }
    
    for step, result in results.items():
        print(f"   {step}: {result}")
    
    print()
    
    success_count = sum(1 for v in results.values() if "✅" in v)
    total_count = len(results)
    
    if success_count == total_count:
        success(f"🎉 Tất cả {total_count} bước đều thành công!")
    else:
        info(f"⚠️  {success_count}/{total_count} bước thành công")
    
    print()
    info("📝 LƯU Ý:")
    info("   - Script chỉ cập nhật mobile và address")
    info("   - Email và password không được cập nhật qua UpdateProfile/UpdateProfileByID")
    info("   - Để đổi password, sử dụng endpoint POST /api/auth/change-password")
    info("   - Email không thể đổi qua bất kỳ endpoint nào (thiết kế bảo mật)")
    print()

if __name__ == "__main__":
    main()

