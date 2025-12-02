#!/usr/bin/env python3
"""Script tự động test các trường hợp lấy rules theo role"""
import json
import sys
import requests
from typing import Dict, Optional, List
from share import (
    info, success, error, login_account, get_config, handle_error_response,
    get_base_url, get_role_names_map, print_rules_list
)

def get_rules_by_role(token: str, role_id_name: str, verbose: bool = True) -> Optional[List]:
    """
    Lấy danh sách rules theo role ID hoặc role name
    
    Args:
        token: JWT token để xác thực
        role_id_name: ID hoặc tên của role (ví dụ: "author", 4, "reader", 2)
        verbose: Nếu True, in ra thông tin chi tiết. Mặc định là True
    
    Returns:
        List các rules hoặc None nếu thất bại
    """
    try:
        if verbose:
            info(f"Đang lấy danh sách rules cho role '{role_id_name}'...")
        
        # URL encode role_id_name để đảm bảo an toàn khi truyền trong URL path
        from urllib.parse import quote
        encoded_role_id_name = quote(str(role_id_name), safe='')
        
        resp = requests.get(
            f"{get_base_url()}/api/rules/role/{encoded_role_id_name}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        # Kiểm tra status code
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, f"lấy danh sách rules cho role '{role_id_name}'")
            except:
                error(f"Response: {resp.text}")
            return None
        
        data = resp.json()
        
        # Kiểm tra response có lỗi không
        if "error" in data:
            handle_error_response(data, f"lấy danh sách rules cho role '{role_id_name}'")
            return None
        
        # Kiểm tra có data không
        if "data" not in data:
            error("Response không hợp lệ:")
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return None
        
        rules = data.get("data", [])
        if verbose:
            success(f"Lấy danh sách rules thành công! Tìm thấy {len(rules)} rules cho role '{role_id_name}'")
        
        return rules
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi khi gọi API: {str(e)}")
        return None
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return None

def main():
    print()
    info("Bắt đầu test các trường hợp lấy rules theo role")
    
    # ==========================================
    # Bước 1: Login với admin@gmail.com
    # ==========================================
    print()
    print("=" * 80)
    info("🔐 Bước 1: Login với admin@gmail.com")
    print("=" * 80)
    
    success_login, admin_token, error_msg = login_account("admin")
    if not success_login:
        error(f"Không thể đăng nhập: {error_msg}")
        sys.exit(1)
    
    # Lấy role names map một lần để tái sử dụng cho tất cả các bước
    info("Đang lấy danh sách roles để map role IDs sang role names...")
    role_names_map = get_role_names_map(admin_token)
    success(f"Đã lấy được {len(role_names_map)} roles")
    
    # ==========================================
    # Bước 2: Gọi GET "/api/rules/role/:id_name" với :id_name = "author"
    # ==========================================
    print()
    print("=" * 80)
    info("📋 Bước 2: Lấy rules cho role 'author'")
    print("=" * 80)
    
    rules_author = get_rules_by_role(admin_token, "author", verbose=True)
    if rules_author is not None:
        print_rules_list(admin_token, rules_author, f"Danh sách rules cho role 'author'", role_names_map)
    else:
        error("Không thể lấy danh sách rules cho role 'author'")
    
    # ==========================================
    # Bước 3: Gọi GET "/api/rules/role/:id_name" với :id_name = 4
    # ==========================================
    print()
    print("=" * 80)
    info("📋 Bước 3: Lấy rules cho role ID = 4")
    print("=" * 80)
    
    rules_role4 = get_rules_by_role(admin_token, 4, verbose=True)
    if rules_role4 is not None:
        print_rules_list(admin_token, rules_role4, f"Danh sách rules cho role ID = 4", role_names_map)
    else:
        error("Không thể lấy danh sách rules cho role ID = 4")
    
    # ==========================================
    # Bước 4: Gọi GET "/api/rules/role/:id_name" với :id_name = "reader"
    # ==========================================
    print()
    print("=" * 80)
    info("📋 Bước 4: Lấy rules cho role 'reader'")
    print("=" * 80)
    
    rules_reader = get_rules_by_role(admin_token, "reader", verbose=True)
    if rules_reader is not None:
        print_rules_list(admin_token, rules_reader, f"Danh sách rules cho role 'reader'", role_names_map)
    else:
        error("Không thể lấy danh sách rules cho role 'reader'")
    
    # ==========================================
    # Bước 5: Gọi GET "/api/rules/role/:id_name" với :id_name = 2
    # ==========================================
    print()
    print("=" * 80)
    info("📋 Bước 5: Lấy rules cho role ID = 2")
    print("=" * 80)
    
    rules_role2 = get_rules_by_role(admin_token, 2, verbose=True)
    if rules_role2 is not None:
        print_rules_list(admin_token, rules_role2, f"Danh sách rules cho role ID = 2", role_names_map)
    else:
        error("Không thể lấy danh sách rules cho role ID = 2")
    
    # ==========================================
    # Tổng kết
    # ==========================================
    print()
    print("=" * 80)
    success("✅ Hoàn thành tất cả các bước!")
    print("=" * 80)

if __name__ == "__main__":
    main()

