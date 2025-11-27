#!/usr/bin/env python3
"""Script tự động test các trường hợp liên quan đến list users có role"""
import json
import sys
import requests
from typing import Dict, Optional, Tuple
from share import info, success, error, login, get_config, handle_error_response, get_user_detail, get_role_id_by_name, get_base_url

def create_role(token: str, role_id: int, role_name: str, is_system: bool = False) -> Tuple[bool, Optional[int]]:
    """
    Tạo role mới
    
    Args:
        token: JWT token để xác thực
        role_id: ID của role
        role_name: Tên role
        is_system: Có phải system role không
    
    Returns:
        Tuple (success, role_id)
    """
    try:
        info(f"Đang tạo role '{role_name}' với ID={role_id}...")
        resp = requests.post(
            f"{get_base_url()}/api/roles",
            json={"id": role_id, "name": role_name, "is_system": is_system},
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "tạo role")
            return False, None
        
        success(f"Tạo role '{role_name}' thành công!")
        return True, role_id
        
    except Exception as e:
        error(f"Lỗi khi tạo role: {str(e)}")
        return False, None

def assign_role_to_user(token: str, user_id: str, role_id: int) -> bool:
    """
    Gán role cho user
    
    Args:
        token: JWT token để xác thực
        user_id: ID của user
        role_id: ID của role
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        resp = requests.post(
            f"{get_base_url()}/api/users/{user_id}/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "gán role")
            return False
        
        success(f"Gán role thành công cho user {user_id}!")
        return True
        
    except Exception as e:
        error(f"Lỗi khi gán role: {str(e)}")
        return False

def remove_role_from_user(token: str, user_id: str, role_id: int) -> bool:
    """
    Xóa role khỏi user
    
    Args:
        token: JWT token để xác thực
        user_id: ID của user
        role_id: ID của role
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        resp = requests.delete(
            f"{get_base_url()}/api/users/{user_id}/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "xóa role khỏi user")
            return False
        
        success(f"Xóa role khỏi user {user_id} thành công!")
        return True
        
    except Exception as e:
        error(f"Lỗi khi xóa role khỏi user: {str(e)}")
        return False

def list_users_has_role(token: str, role_id_name: str) -> Optional[list]:
    """
    Lấy danh sách users có role
    
    Args:
        token: JWT token để xác thực
        role_id_name: ID hoặc tên của role
    
    Returns:
        List các users hoặc None nếu thất bại
    """
    try:
        info(f"Đang lấy danh sách users có role '{role_id_name}'...")
        resp = requests.get(
            f"{get_base_url()}/api/roles/{role_id_name}/users",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, "lấy danh sách users có role")
            except:
                error(f"Response: {resp.text}")
            return None
        
        data = resp.json()
        
        if "error" in data:
            handle_error_response(data, "lấy danh sách users có role")
            return None
        
        if "data" not in data:
            error("Response không hợp lệ:")
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return None
        
        users = data.get("data", [])
        success(f"Lấy danh sách users thành công! Tìm thấy {len(users)} users")
        
        return users
        
    except Exception as e:
        error(f"Lỗi khi lấy danh sách users: {str(e)}")
        return None

def delete_role(token: str, role_id: int) -> bool:
    """
    Xóa role
    
    Args:
        token: JWT token để xác thực
        role_id: ID của role
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        info(f"Đang xóa role với ID={role_id}...")
        resp = requests.delete(
            f"{get_base_url()}/api/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "xóa role")
            return False
        
        success(f"Xóa role ID {role_id} thành công!")
        return True
        
    except Exception as e:
        error(f"Lỗi khi xóa role: {str(e)}")
        return False

def print_users_list(users: list, title: str = "Danh sách users"):
    """In danh sách users với roles"""
    print()
    print("=" * 60)
    info(f"📋 {title}")
    print("=" * 60)
    
    if not users:
        info("Không có user nào")
        return
    
    for idx, user in enumerate(users, 1):
        user_id = user.get("id", "N/A")
        email = user.get("email", "N/A")
        
        # Lấy roles từ user object (đã có sẵn trong response)
        roles = user.get("roles", [])
        roles_list = [role.get("name", "") for role in roles if isinstance(role, dict) and role.get("name")]
        
        # Format roles thành chuỗi
        roles_str = ", ".join(roles_list) if roles_list else "N/A"
        
        print(f"{idx}. User ID: {user_id}, Email: {email}, Roles: {roles_str}")
    
    print()

def get_user_id_by_email(token: str, email: str) -> Optional[str]:
    """
    Lấy user_id từ email
    
    Args:
        token: JWT token để xác thực
        email: Email của user
    
    Returns:
        user_id hoặc None nếu không tìm thấy
    """
    user_detail = get_user_detail(token, email)
    if not user_detail or "user" not in user_detail:
        error(f"Không thể lấy user_id của {email}. Có thể user chưa tồn tại.")
        return None
    
    user_id = user_detail["user"].get("id")
    if not user_id:
        error(f"Không thể lấy user_id của {email} từ response.")
        return None
    
    return user_id

def main():
    print()
    info("Bắt đầu test các trường hợp liên quan đến list users có role")
    
    # ==========================================
    # Bước 1: Login admin
    # ==========================================
    print()
    print("=" * 60)
    info("🔐 Bước 1: Login admin")
    print("=" * 60)
    
    config = get_config()
    admin_token, admin_user = login(config["admin_email"], config["admin_password"])
    
    # ==========================================
    # Bước 2: Tạo role 'puma'
    # ==========================================
    print()
    print("=" * 60)
    info("➕ Bước 2: Tạo role 'puma'")
    print("=" * 60)
    
    # Tìm role_id trống (có thể dùng một số lớn để tránh conflict)
    # Hoặc có thể kiểm tra role_id hiện có, nhưng để đơn giản dùng ID cố định
    puma_role_id = 200  # Có thể thay đổi nếu cần
    puma_role_name = "puma"
    
    create_success, created_role_id = create_role(admin_token, puma_role_id, puma_role_name, is_system=False)
    if not create_success:
        error("Không thể tạo role 'puma'")
        sys.exit(1)
    
    # Lấy role_id thực tế (có thể khác nếu server tự động assign ID)
    puma_role_id = get_role_id_by_name(admin_token, puma_role_name)
    if not puma_role_id:
        error("Không tìm thấy role 'puma' sau khi tạo")
        sys.exit(1)
    
    info(f"Role 'puma' có ID: {puma_role_id}")
    
    # ==========================================
    # Bước 3: Add role 'puma' cho các users
    # ==========================================
    print()
    print("=" * 60)
    info("👥 Bước 3: Add role 'puma' cho các users")
    print("=" * 60)
    
    user_emails = ["author1@gmail.com", "author2@gmail.com", "bob@gmail.com"]
    user_ids = {}
    
    for email in user_emails:
        info(f"Đang lấy user_id cho {email}...")
        user_id = get_user_id_by_email(admin_token, email)
        if not user_id:
            error(f"Không thể lấy user_id của {email}, bỏ qua...")
            continue
        
        user_ids[email] = user_id
        info(f"User {email} có ID: {user_id}")
        
        # Add role
        if assign_role_to_user(admin_token, user_id, puma_role_id):
            success(f"Đã gán role 'puma' cho {email}")
        else:
            error(f"Không thể gán role 'puma' cho {email}")
    
    # ==========================================
    # Bước 4: Lấy danh sách users có role 'puma'
    # ==========================================
    print()
    print("=" * 60)
    info("📋 Bước 4: Lấy danh sách users có role 'puma'")
    print("=" * 60)
    
    users_with_puma = list_users_has_role(admin_token, puma_role_name)
    if users_with_puma is not None:
        print_users_list(users_with_puma, f"Danh sách users có role '{puma_role_name}'")
    else:
        error("Không thể lấy danh sách users có role 'puma'")
    
    # ==========================================
    # Bước 5: Remove role 'puma' khỏi các users
    # ==========================================
    print()
    print("=" * 60)
    info("➖ Bước 5: Remove role 'puma' khỏi các users")
    print("=" * 60)
    
    for email, user_id in user_ids.items():
        info(f"Đang xóa role 'puma' khỏi {email}...")
        remove_role_from_user(admin_token, user_id, puma_role_id)
    
    # ==========================================
    # Bước 6: Lấy lại danh sách users có role 'puma'
    # ==========================================
    print()
    print("=" * 60)
    info("📋 Bước 6: Lấy lại danh sách users có role 'puma'")
    print("=" * 60)
    
    users_with_puma_after = list_users_has_role(admin_token, puma_role_name)
    if users_with_puma_after is not None:
        print_users_list(users_with_puma_after, f"Danh sách users có role '{puma_role_name}' (sau khi xóa)")
    else:
        error("Không thể lấy danh sách users có role 'puma'")
    
    # ==========================================
    # Bước 7: Xóa role 'puma'
    # ==========================================
    print()
    print("=" * 60)
    info("🗑️  Bước 7: Xóa role 'puma'")
    print("=" * 60)
    
    if delete_role(admin_token, puma_role_id):
        success("Đã xóa role 'puma' thành công!")
    else:
        error("Không thể xóa role 'puma'")
    
    # ==========================================
    # Tổng kết
    # ==========================================
    print()
    print("=" * 60)
    info("✅ Hoàn thành tất cả các bước!")
    print("=" * 60)

if __name__ == "__main__":
    main()

