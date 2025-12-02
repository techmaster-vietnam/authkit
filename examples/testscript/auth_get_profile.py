#!/usr/bin/env python3
"""Script test các endpoint profile:
- GET /api/user/profile - Xem profile của chính mình
- GET /api/user/:id - Xem profile theo identifier (id, email, mobile) - chỉ admin/super_admin

Script này kết hợp chức năng từ user_get_profile.py và user_get_detail.py
"""
import json
import sys
from typing import Dict, Optional

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

from share import (
    info, success, error, get_config, get_user_detail,
    print_section, login_with_error_handling, get_profile, get_profile_by_identifier
)


def test_profile_by_identifier(admin_token: str, identifier: str, identifier_type: str, 
                               skip_if_none: bool = False) -> bool:
    """
    Test lấy profile bằng identifier
    
    Args:
        admin_token: Admin token để xác thực
        identifier: ID, email hoặc mobile để test
        identifier_type: Loại identifier ("ID", "Email", "Mobile")
        skip_if_none: Nếu True, bỏ qua test nếu identifier là None (không báo lỗi)
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    if not identifier:
        if skip_if_none:
            info(f"Không có {identifier_type} để test, bỏ qua")
            print()
            return False
        else:
            error(f"Không có {identifier_type} để test")
            return False
    
    info(f"Đang test với {identifier_type}: {identifier}")
    result = get_profile_by_identifier(admin_token, identifier)
    
    if not result:
        error(f"❌ Không tìm thấy user với {identifier_type}: {identifier}")
        return False
    
    print()
    return True


def test_profile_endpoints():
    """Test các endpoint profile với các test cases chi tiết"""
    config = get_config()
    
    print()
    info("Bắt đầu test các endpoint profile")
    print()
    
    # ============================================================
    # Test 1: Bob login và xem profile của chính mình
    # ============================================================
    print_section("Test 1: Bob xem profile của chính mình")
    
    bob_token = login_with_error_handling("bob@gmail.com", "123456", "Bob")
    print()
    
    # Lấy profile của chính mình (không verbose để tránh in trùng)
    bob_profile = get_profile(bob_token, verbose=False)
    if not bob_profile:
        error("Không thể lấy profile của Bob")
        sys.exit(1)
    
    # Lưu thông tin của Bob để dùng cho test sau
    bob_id = bob_profile.get('id')
    bob_email = bob_profile.get('email')
    bob_mobile = bob_profile.get('mobile')
    
    # Đăng nhập admin để xem profile của Bob với roles
    admin_token = login_with_error_handling(
        "admin@gmail.com", 
        config.get("admin_password", "123456"),
        "admin"
    )
    print()
    
    # Hiển thị profile của Bob với roles
    if bob_id:
        info(f"Đang lấy thông tin profile của Bob (ID: {bob_id})...")
        if not get_profile_by_identifier(admin_token, bob_id):
            error("Không thể lấy profile của Bob")
        print()
    else:
        error("Không có ID của Bob")
    
    # ============================================================
    # Test 2-4: Admin xem profile của Bob bằng các identifier khác nhau
    # ============================================================
    test_cases = [
        ("Test 2: Admin xem profile của Bob bằng ID", bob_id, "ID", False),
        ("Test 3: Admin xem profile của Bob bằng Email", bob_email, "Email", False),
        ("Test 4: Admin xem profile của Bob bằng Mobile", bob_mobile, "Mobile", True),
    ]
    
    for test_title, identifier, identifier_type, skip_if_none in test_cases:
        print_section(test_title)
        test_profile_by_identifier(admin_token, identifier, identifier_type, skip_if_none)
    
    # ============================================================
    # Kết thúc
    # ============================================================
    print("=" * 80)
    success("Test các endpoint profile hoàn thành!")
    print("=" * 80)


def batch_get_user_profiles():
    """Lấy thông tin chi tiết của nhiều user (batch processing)"""
    # Danh sách các user cần lấy thông tin (có thể là email, ID hoặc mobile)
    user_identifiers = [
        "editor@gmail.com",
        "author1@gmail.com",
        "bob@gmail.com",
        "JOQkg6Ao4S7V"
    ]
    
    print()
    info("Bắt đầu batch lấy thông tin chi tiết các user")
    print()
    
    # Login một lần với admin account để lấy token
    info("Đang đăng nhập với admin account...")
    token = login_with_error_handling("admin@gmail.com", "123456", "admin")
    print()
    
    # Lặp qua từng user và lấy thông tin (dùng chung token)
    for idx, identifier in enumerate(user_identifiers, 1):
        print("=" * 80)
        info(f"[{idx}/{len(user_identifiers)}] Xử lý user: {identifier}")
        print("=" * 80)
        print()
        
        user_detail = get_user_detail(token, identifier)
        
        if user_detail:
            success(f"Hoàn thành lấy thông tin cho: {identifier}")
        else:
            error(f"Không thể lấy thông tin cho: {identifier}")
        
        print()
    
    print("=" * 80)
    success("Batch processing hoàn thành!")
    print("=" * 80)


def main():
    """Hàm main chạy tất cả test cases"""
    print()
    print("=" * 80)
    info("Script test các endpoint profile")
    print("=" * 80)
    print()
    
    # Chạy tất cả test cases
    # 1. Test các endpoint profile với test cases chi tiết
    test_profile_endpoints()
    
    print()
    
    # 2. Batch lấy thông tin nhiều user
    batch_get_user_profiles()
    
    print()
    print("=" * 80)
    success("Tất cả test cases đã hoàn thành!")
    print("=" * 80)


if __name__ == "__main__":
    main()

