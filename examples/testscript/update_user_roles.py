#!/usr/bin/env python3
"""Script tự động test cập nhật roles cho user"""
import sys
import json
import requests
from share import (
    info, success, error, login, get_config, 
    create_role, get_user_detail, update_user_roles,
    get_role_id_by_name, handle_error_response, get_base_url
)

def verify_roles(user_detail, expected_role_names: list) -> bool:
    """
    Kiểm tra danh sách roles của user có khớp với danh sách mong đợi không
    
    Args:
        user_detail: Dictionary chứa thông tin user detail
        expected_role_names: Danh sách role names mong đợi (ví dụ: ["author", "reader", "tiger"])
    
    Returns:
        True nếu khớp, False nếu không khớp
    """
    if not user_detail or "roles" not in user_detail:
        if not expected_role_names:
            return True
        return False
    
    roles = user_detail.get("roles", [])
    actual_role_names = [role.get('role_name') for role in roles if role.get('role_name')]
    
    # Sắp xếp để so sánh
    actual_sorted = sorted(actual_role_names)
    expected_sorted = sorted(expected_role_names)
    
    if actual_sorted == expected_sorted:
        success(f"✅ Danh sách roles khớp với mong đợi: {expected_role_names}")
        return True
    else:
        error(f"❌ Danh sách roles không khớp!")
        error(f"   Mong đợi: {expected_role_names}")
        error(f"   Thực tế: {actual_role_names}")
        return False

def print_user_roles(user_detail, user_email: str = None):
    """
    In danh sách roles của user (không in user detail)
    
    Args:
        user_detail: Dictionary chứa thông tin user detail
        user_email: Email của user (để hiển thị, optional)
    """
    if not user_detail or "roles" not in user_detail:
        if user_email:
            info(f"User {user_email} không có role nào")
        else:
            info("Không có role nào")
        return
    
    roles = user_detail.get("roles", [])
    
    if user_email:
        info(f"Danh sách roles của {user_email}:")
    else:
        info("Danh sách roles:")
    
    if roles:
        for idx, role in enumerate(roles, 1):
            role_id = role.get('role_id', 'N/A')
            role_name = role.get('role_name', 'N/A')
            print(f"  {idx}. Role ID: {role_id}, Role Name: {role_name}")
    else:
        info("  Không có role nào")
    print()

def delete_role(token: str, role_name: str) -> bool:
    """
    Xóa role theo name
    
    Args:
        token: JWT token để xác thực
        role_name: Tên role cần xóa
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        info(f"Đang tìm role_id từ role_name '{role_name}'...")
        role_id = get_role_id_by_name(token, role_name)
        if role_id is None:
            error(f"Không tìm thấy role với name '{role_name}'")
            return False
        info(f"Tìm thấy role_id: {role_id}")
        
        # Xóa role
        print()
        info(f"Đang xóa role '{role_name}' (ID: {role_id})...")
        
        resp = requests.delete(
            f"{get_base_url()}/api/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        print()
        info("Response từ server:")
        delete_data = resp.json()
        print(json.dumps(delete_data, indent=2, ensure_ascii=False))
        
        # Kiểm tra lỗi
        if resp.status_code >= 400 or "error" in delete_data:
            handle_error_response(delete_data, "xóa role")
            return False
        
        # Kiểm tra thành công
        if "message" in delete_data:
            message = delete_data.get("message")
            success(message)
        else:
            if 200 <= resp.status_code < 300:
                success(f"Xóa role '{role_name}' thành công!")
            else:
                error(f"Response không chứa message, có thể có lỗi (HTTP {resp.status_code})")
                return False
        
        return True
        
    except Exception as e:
        error(f"Lỗi khi xóa role: {str(e)}")
        return False

def main():
    print()
    info("Bắt đầu script cập nhật roles cho user")
    
    # ==========================================
    # Bước 1: Login với admin@gmail.com
    # ==========================================
    print()
    print("=" * 60)
    info("🔐 Bước 1: Login với admin@gmail.com")
    print("=" * 60)
    
    config = get_config()
    token, user = login(config["admin_email"], config["admin_password"])
    
    # ==========================================
    # Bước 2: Tạo 3 roles mới
    # ==========================================
    print()
    print("=" * 60)
    info("➕ Bước 2: Tạo 3 roles mới")
    print("=" * 60)
    
    roles_to_create = [
        (200, "tiger"),
        (300, "puma"),
        (400, "dragon"),
    ]
    
    for role_id, role_name in roles_to_create:
        print()
        info(f"Tạo role: ID={role_id}, Name={role_name}")
        if not create_role(token, role_id, role_name, is_system=False):
            error(f"Không thể tạo role {role_name}")
            sys.exit(1)
    
    # ==========================================
    # Bước 3: Lấy thông tin chi tiết user bob@gmail.com
    # ==========================================
    print()
    print("=" * 60)
    info("👤 Bước 3: Lấy thông tin chi tiết user bob@gmail.com")
    print("=" * 60)
    
    bob_email = "bob@gmail.com"
    bob_detail = get_user_detail(token, bob_email, verbose=False)
    
    if not bob_detail or "user" not in bob_detail:
        error(f"Không thể lấy thông tin user {bob_email}. Có thể user chưa tồn tại.")
        sys.exit(1)
    
    bob_user_id = bob_detail["user"].get("id")
    if not bob_user_id:
        error(f"Không thể lấy user_id của {bob_email} từ response.")
        sys.exit(1)
    
    # ==========================================
    # Bước 4: Cập nhật roles cho bob thành ["author", "reader", "tiger"]
    # ==========================================
    print()
    print("=" * 60)
    info("🔄 Bước 4: Cập nhật roles cho bob thành ['author', 'reader', 'tiger']")
    print("=" * 60)
    
    new_roles_1 = ["author", "reader", "tiger"]
    success_1, _ = update_user_roles(token, bob_user_id, new_roles_1)
    
    if not success_1:
        error("Không thể cập nhật roles cho bob")
        sys.exit(1)
    
    # Lấy lại thông tin user để kiểm tra
    bob_detail_1 = get_user_detail(token, bob_email, verbose=False)
    
    if bob_detail_1:
        print_user_roles(bob_detail_1, bob_email)
        verify_roles(bob_detail_1, new_roles_1)
    
    # ==========================================
    # Bước 5: Cập nhật roles cho bob thành ["tiger", "puma", "dragon"]
    # ==========================================
    print()
    print("=" * 60)
    info("🔄 Bước 5: Cập nhật roles cho bob thành ['tiger', 'puma', 'dragon']")
    print("=" * 60)
    
    new_roles_2 = ["tiger", "puma", "dragon"]
    success_2, _ = update_user_roles(token, bob_user_id, new_roles_2)
    
    if not success_2:
        error("Không thể cập nhật roles cho bob")
        sys.exit(1)
    
    # Lấy lại thông tin user để kiểm tra
    bob_detail_2 = get_user_detail(token, bob_email, verbose=False)
    
    if bob_detail_2:
        print_user_roles(bob_detail_2, bob_email)
        verify_roles(bob_detail_2, new_roles_2)
    
    # ==========================================
    # Bước 6: Xóa 3 roles mới tạo: "tiger", "puma", "dragon"
    # ==========================================
    print()
    print("=" * 60)
    info("🗑️  Bước 6: Xóa 3 roles mới tạo: 'tiger', 'puma', 'dragon'")
    print("=" * 60)
    
    roles_to_delete = ["tiger", "puma", "dragon"]
    
    for role_name in roles_to_delete:
        print()
        info(f"Đang xóa role: {role_name}")
        if not delete_role(token, role_name):
            error(f"Không thể xóa role {role_name}")
            sys.exit(1)
    
    # ==========================================
    # Tổng kết
    # ==========================================
    print()
    print("=" * 60)
    success("✅ Hoàn thành tất cả các bước!")
    print("=" * 60)

if __name__ == "__main__":
    main()

