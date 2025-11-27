#!/usr/bin/env python3
"""Script tự động login, tạo role, liệt kê roles, xóa role và liệt kê lại"""
import json
import sys
import requests
from share import info, success, error, login, get_config, handle_error_response, get_base_url, get_role_id_by_name, create_role

def list_roles(token: str) -> list:
    """
    Liệt kê tất cả roles
    
    Args:
        token: JWT token để xác thực
    
    Returns:
        List các roles hoặc empty list nếu thất bại
    """
    try:
        info("Đang lấy danh sách roles...")
        resp = requests.get(
            f"{get_base_url()}/api/roles",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, "lấy danh sách roles")
            except:
                error(f"Response: {resp.text}")
            return []
        
        data = resp.json()
        
        if "error" in data:
            handle_error_response(data, "lấy danh sách roles")
            return []
        
        if "data" not in data:
            error("Response không hợp lệ:")
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return []
        
        roles = data.get("data", [])
        success(f"Lấy danh sách roles thành công! Tìm thấy {len(roles)} roles")
        return roles
        
    except Exception as e:
        error(f"Lỗi khi lấy danh sách roles: {str(e)}")
        return []

def print_roles_list(roles: list, title: str = "Danh sách roles"):
    """In danh sách roles"""
    print()
    print("=" * 60)
    info(f"📋 {title}")
    print("=" * 60)
    
    if not roles:
        info("Không có role nào")
        return
    
    for idx, role in enumerate(roles, 1):
        role_id = role.get("id", "N/A")
        role_name = role.get("name", "N/A")
        is_system = role.get("is_system", False)
        system_str = "System" if is_system else "User"
        print(f"{idx}. Role ID: {role_id}, Name: {role_name}, Type: {system_str}")
    
    print()

def delete_role(token: str, role_id: int = None, role_name: str = None) -> bool:
    """
    Xóa role theo ID hoặc name
    
    Args:
        token: JWT token để xác thực
        role_id: ID của role (ưu tiên)
        role_name: Tên role (nếu không có role_id)
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        # Nếu không có role_id, tìm từ role_name
        if role_id is None:
            if role_name is None:
                error("Cần cung cấp role_id hoặc role_name để xóa role")
                return False
            
            info(f"Đang tìm role_id từ role_name '{role_name}'...")
            role_id = get_role_id_by_name(token, role_name)
            if role_id is None:
                error(f"Không tìm thấy role với name '{role_name}'")
                return False
            info(f"Tìm thấy role_id: {role_id}")
        
        # Xóa role
        print()
        info("Đang xóa role...")
        info(f"  - Role ID: {role_id}")
        if role_name:
            info(f"  - Role Name: {role_name}")
        
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
            print()
            info(f"Role ID {role_id} đã được xóa thành công!")
            info("Stored procedure đã tự động:")
            info("  - Xóa tất cả bản ghi trong user_roles có role_id = " + str(role_id))
            info("  - Xóa role_id khỏi mảng roles trong bảng rules")
            info("  - Xóa bản ghi trong bảng roles")
        else:
            # Kiểm tra HTTP status code
            if 200 <= resp.status_code < 300:
                success(f"Xóa role thành công (HTTP {resp.status_code})")
            else:
                error(f"Response không chứa message, có thể có lỗi (HTTP {resp.status_code})")
                return False
        
        return True
        
    except Exception as e:
        error(f"Lỗi khi xóa role: {str(e)}")
        return False

def main():
    print()
    info("Bắt đầu script tạo và xóa role")
    
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
    # Bước 2: Tạo role id:500, name: "dragon"
    # ==========================================
    print()
    print("=" * 60)
    info("➕ Bước 2: Tạo role id:500, name: 'dragon'")
    print("=" * 60)
    
    role_id = 500
    role_name = "dragon"
    
    if not create_role(token, role_id, role_name, is_system=False):
        error("Không thể tạo role")
        sys.exit(1)
    
    # ==========================================
    # Bước 3: Liệt kê danh sách role sau khi tạo thành công
    # ==========================================
    print()
    print("=" * 60)
    info("📋 Bước 3: Liệt kê danh sách role sau khi tạo thành công")
    print("=" * 60)
    
    roles_after_create = list_roles(token)
    print_roles_list(roles_after_create, "Danh sách roles sau khi tạo")
    
    # ==========================================
    # Bước 4: Xóa role có id =500 hoặc name ="dragon"
    # ==========================================
    print()
    print("=" * 60)
    info("🗑️  Bước 4: Xóa role có id=500 hoặc name='dragon'")
    print("=" * 60)
    
    if not delete_role(token, role_id=500, role_name="dragon"):
        error("Không thể xóa role")
        sys.exit(1)
    
    # ==========================================
    # Bước 5: Liệt kê danh sách role sau khi xóa role thành công
    # ==========================================
    print()
    print("=" * 60)
    info("📋 Bước 5: Liệt kê danh sách role sau khi xóa role thành công")
    print("=" * 60)
    
    roles_after_delete = list_roles(token)
    print_roles_list(roles_after_delete, "Danh sách roles sau khi xóa")
    
    # ==========================================
    # Tổng kết
    # ==========================================
    print()
    print("=" * 60)
    success("✅ Hoàn thành tất cả các bước!")
    print("=" * 60)

if __name__ == "__main__":
    main()

