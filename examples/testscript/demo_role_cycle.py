#!/usr/bin/env python3
"""Script demo chu trình quản lý role: tạo role, gán cho user, hiển thị, và xóa"""
import json
import sys
import requests
from share import (
    info, success, error, login, get_config, handle_error_response, 
    get_base_url, get_user_detail, get_role_id_by_name, get_user_roles
)

def create_role(token: str, role_name: str, role_id: int = None, is_system: bool = False) -> dict:
    """
    Tạo role mới
    
    Args:
        token: JWT token để xác thực
        role_name: Tên role cần tạo
        role_id: ID của role (optional, nếu None sẽ để server tự động)
        is_system: Có phải system role không
    
    Returns:
        Dictionary chứa thông tin role đã tạo, hoặc None nếu thất bại
    """
    try:
        payload = {"name": role_name, "is_system": is_system}
        if role_id is not None:
            payload["id"] = role_id
        
        info(f"Đang tạo role '{role_name}'...")
        resp = requests.post(
            f"{get_base_url()}/api/roles",
            json=payload,
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "tạo role")
            return None
        
        success(f"Tạo role '{role_name}' thành công!")
        return resp_data.get("data", {})
        
    except Exception as e:
        error(f"Lỗi khi tạo role: {str(e)}")
        return None

def add_role_to_user(token: str, user_id: str, role_id: int) -> bool:
    """
    Thêm role cho user
    
    Args:
        token: JWT token để xác thực
        user_id: ID của user
        role_id: ID của role
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        info(f"Đang thêm role ID {role_id} cho user {user_id}...")
        resp = requests.post(
            f"{get_base_url()}/api/users/{user_id}/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "thêm role cho user")
            return False
        
        success("Thêm role cho user thành công!")
        return True
        
    except Exception as e:
        error(f"Lỗi khi thêm role cho user: {str(e)}")
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
        info(f"Đang xóa role ID {role_id} khỏi user {user_id}...")
        resp = requests.delete(
            f"{get_base_url()}/api/users/{user_id}/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "xóa role khỏi user")
            return False
        
        success("Xóa role khỏi user thành công!")
        return True
        
    except Exception as e:
        error(f"Lỗi khi xóa role khỏi user: {str(e)}")
        return False

def delete_role(token: str, role_id: int) -> bool:
    """
    Xóa role khỏi database
    
    Args:
        token: JWT token để xác thực
        role_id: ID của role cần xóa
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        info(f"Đang xóa role ID {role_id} khỏi database...")
        resp = requests.delete(
            f"{get_base_url()}/api/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "xóa role khỏi database")
            return False
        
        success("Xóa role khỏi database thành công!")
        if "message" in resp_data:
            info(f"Chi tiết: {resp_data.get('message')}")
        return True
        
    except Exception as e:
        error(f"Lỗi khi xóa role khỏi database: {str(e)}")
        return False

def main():
    """Hàm main để demo chu trình quản lý role"""
    print()
    print("=" * 80)
    info("🚀 Bắt đầu demo chu trình quản lý role")
    print("=" * 80)
    print()
    
    # ==========================================
    # Bước 1: Login với admin
    # ==========================================
    print("=" * 80)
    info("Bước 1: Đăng nhập với admin account")
    print("=" * 80)
    
    config = get_config()
    admin_token, admin_user = login(config["admin_email"], config["admin_password"])
    
    if not admin_token:
        error("Không thể đăng nhập với admin account")
        sys.exit(1)
    
    print()
    
    # ==========================================
    # Bước 2: Tạo role "tiger"
    # ==========================================
    print("=" * 80)
    info("Bước 2: Tạo role 'tiger'")
    print("=" * 80)
    
    role_name = "tiger"
    role_data = create_role(admin_token, role_name, 100, is_system=False)
    
    if not role_data:
        error("Không thể tạo role 'tiger'")
        sys.exit(1)
    
    tiger_role_id = role_data.get("id")
    if not tiger_role_id:
        error("Không thể lấy role_id từ response")
        sys.exit(1)
    
    info(f"Role 'tiger' đã được tạo với ID: {tiger_role_id}")
    print()
    
    # ==========================================
    # Bước 3: Add role "tiger" vào user "bob@gmail.com"
    # ==========================================
    print("=" * 80)
    info("Bước 3: Thêm role 'tiger' cho user 'bob@gmail.com'")
    print("=" * 80)
    
    bob_email = "bob@gmail.com"
    
    # Lấy user_id của bob
    user_detail = get_user_detail(admin_token, bob_email)
    if not user_detail or "user" not in user_detail:
        error(f"Không thể lấy thông tin user '{bob_email}'. Có thể user chưa tồn tại.")
        sys.exit(1)
    
    bob_user_id = user_detail["user"].get("id")
    if not bob_user_id:
        error(f"Không thể lấy user_id của '{bob_email}' từ response.")
        sys.exit(1)
    
    info(f"User ID của '{bob_email}': {bob_user_id}")
    
    # Thêm role
    if not add_role_to_user(admin_token, bob_user_id, tiger_role_id):
        error("Không thể thêm role 'tiger' cho user")
        sys.exit(1)
    
    print()
    
    # ==========================================
    # Bước 4: Hiển thị thông tin user bob@gmail.com (bao gồm danh sách roles)
    # ==========================================
    print("=" * 80)
    info("Bước 4: Hiển thị thông tin chi tiết của user 'bob@gmail.com'")
    print("=" * 80)
    print()
    
    user_detail = get_user_detail(admin_token, bob_email)
    
    if not user_detail:
        error("Không thể lấy thông tin chi tiết user")
        sys.exit(1)
    
    print()
    
    # ==========================================
    # Bước 5: Remove role "tiger" from user bob
    # ==========================================
    print("=" * 80)
    info("Bước 5: Xóa role 'tiger' khỏi user 'bob@gmail.com'")
    print("=" * 80)
    
    if not remove_role_from_user(admin_token, bob_user_id, tiger_role_id):
        error("Không thể xóa role 'tiger' khỏi user")
        sys.exit(1)
    
    print()
    
    # ==========================================
    # Bước 6: In ra danh sách roles của user bob
    # ==========================================
    print("=" * 80)
    info("Bước 6: Hiển thị danh sách roles của user 'bob@gmail.com'")
    print("=" * 80)
    print()
    
    # Sử dụng hàm get_user_roles() từ share.py thay vì tự parse
    roles = get_user_roles(admin_token, bob_email)
    
    if roles is not None:
        print()
        print("=" * 80)
        success("Danh sách roles của user bob:")
        print("=" * 80)
        info(f"Email: {bob_email}")
        info(f"Số lượng roles: {len(roles)}")
        
        if roles:
            info("Danh sách roles:")
            for role_id, role_name in roles:
                print(f"  - Role ID: {role_id}, Role Name: {role_name}")
        else:
            info("User không có role nào")
        
        print("=" * 80)
    else:
        error("Không thể lấy danh sách roles của user")
        sys.exit(1)
    
    print()
    
    # ==========================================
    # Bước 7: Xóa role "tiger" khỏi database
    # ==========================================
    print("=" * 80)
    info("Bước 7: Xóa role 'tiger' khỏi database")
    print("=" * 80)
    
    if not delete_role(admin_token, tiger_role_id):
        error("Không thể xóa role 'tiger' khỏi database")
        sys.exit(1)
    
    print()
    
    # ==========================================
    # Tổng kết
    # ==========================================
    print("=" * 80)
    success("✅ Demo chu trình quản lý role hoàn thành!")
    print("=" * 80)
    print()
    info("Các bước đã thực hiện:")
    info("  1. ✅ Đăng nhập với admin account")
    info("  2. ✅ Tạo role 'tiger'")
    info("  3. ✅ Thêm role 'tiger' cho user 'bob@gmail.com'")
    info("  4. ✅ Hiển thị thông tin chi tiết user (bao gồm danh sách roles)")
    info("  5. ✅ Xóa role 'tiger' khỏi user 'bob@gmail.com'")
    info("  6. ✅ Hiển thị danh sách roles của user 'bob@gmail.com'")
    info("  7. ✅ Xóa role 'tiger' khỏi database")
    print()

if __name__ == "__main__":
    main()

