#!/usr/bin/env python3
"""Script tự động test các trường hợp lọc rules"""
import sys
from share import (
    info, success, error, login, get_config, 
    filter_rules, print_rules_list, get_role_names_map
)

def main():
    print()
    info("Bắt đầu test các trường hợp lọc rules")
    
    # ==========================================
    # Bước 1: Login với admin@gmail.com
    # ==========================================
    print()
    print("=" * 60)
    info("🔐 Bước 1: Login với admin@gmail.com")
    print("=" * 60)
    
    config = get_config()
    token, user = login(config["admin_email"], config["admin_password"])
    
    # Lấy role names map một lần để tái sử dụng cho tất cả các bước
    info("Đang lấy danh sách roles để map role IDs sang role names...")
    role_names_map = get_role_names_map(token)
    success(f"Đã lấy được {len(role_names_map)} roles")
    
    # ==========================================
    # Bước 2: Liệt kê tất cả các rules (không có tham số lọc)
    # ==========================================
    print()
    print("=" * 60)
    info("📋 Bước 2: Liệt kê tất cả các rules (không có tham số lọc)")
    print("=" * 60)
    
    all_rules = filter_rules(token)
    print_rules_list(token, all_rules, "Tất cả các rules", role_names_map)
    
    # ==========================================
    # Bước 3: Lọc các rule có method là "GET"
    # ==========================================
    print()
    print("=" * 60)
    info("🔍 Bước 3: Lọc các rule có method là 'GET'")
    print("=" * 60)
    
    get_rules = filter_rules(token, method="GET")
    print_rules_list(token, get_rules, "Rules có method GET", role_names_map)
    
    # ==========================================
    # Bước 4: Lọc các rule có method là "PUT"
    # ==========================================
    print()
    print("=" * 60)
    info("🔍 Bước 4: Lọc các rule có method là 'PUT'")
    print("=" * 60)
    
    put_rules = filter_rules(token, method="PUT")
    print_rules_list(token, put_rules, "Rules có method PUT", role_names_map)
    
    # ==========================================
    # Bước 5: Lọc các rule có method là "POST" và path chứa "blog"
    # ==========================================
    print()
    print("=" * 60)
    info("🔍 Bước 5: Lọc các rule có method là 'POST' và path chứa 'blog'")
    print("=" * 60)
    
    post_blog_rules = filter_rules(token, method="POST", path="blog")
    print_rules_list(token, post_blog_rules, "Rules có method POST và path chứa 'blog'", role_names_map)
    
    # ==========================================
    # Bước 6: Lọc các rule có method là "GET" và type là "PUBLIC"
    # ==========================================
    print()
    print("=" * 60)
    info("🔍 Bước 6: Lọc các rule có method là 'GET' và type là 'PUBLIC'")
    print("=" * 60)
    
    get_public_rules = filter_rules(token, method="GET", type_param="PUBLIC")
    print_rules_list(token, get_public_rules, "Rules có method GET và type PUBLIC", role_names_map)
    
    # ==========================================
    # Bước 7: Lọc các rule có method là "POST" và fixed = true
    # ==========================================
    print()
    print("=" * 60)
    info("🔍 Bước 7: Lọc các rule có method là 'POST' và fixed = true")
    print("=" * 60)
    
    post_fixed_rules = filter_rules(token, method="POST", fixed=True)
    print_rules_list(token, post_fixed_rules, "Rules có method POST và fixed = true", role_names_map)
    
    # ==========================================
    # Tổng kết
    # ==========================================
    print()
    print("=" * 60)
    success("✅ Hoàn thành tất cả các bước!")
    print("=" * 60)

if __name__ == "__main__":
    main()

