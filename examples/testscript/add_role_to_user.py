#!/usr/bin/env python3
"""Script tự động test các trường hợp gán role cho user"""
import json
import sys
import requests
from typing import Dict, Optional, Tuple
from share import info, success, error, login, get_config, handle_error_response, get_user_detail, get_role_id_by_name, get_base_url

def assign_role_to_user(
    token: str,
    user_id: str,
    role_id: int,
    expected_success: bool = True
) -> Tuple[bool, Dict]:
    """
    Gán role cho user
    
    Args:
        token: JWT token để xác thực
        user_id: ID của user
        role_id: ID của role
        expected_success: True nếu mong đợi thành công, False nếu mong đợi lỗi
    
    Returns:
        Tuple (success, response_data)
    """
    try:
        resp = requests.post(
            f"{get_base_url()}/api/users/{user_id}/roles/{role_id}",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        # Kiểm tra kết quả
        is_success = resp.status_code < 400 and "error" not in resp_data
        
        if expected_success:
            if is_success:
                success("Gán role thành công!")
                return True, resp_data
            else:
                error("Gán role thất bại (mong đợi thành công)")
                handle_error_response(resp_data, "gán role")
                return False, resp_data
        else:
            if not is_success:
                success("Gán role thất bại như mong đợi (lỗi nghiệp vụ/authorization)")
                handle_error_response(resp_data, "gán role")
                return True, resp_data
            else:
                error("Gán role thành công (nhưng mong đợi lỗi)")
                return False, resp_data
                
    except Exception as e:
        error(f"Lỗi khi gán role: {str(e)}")
        return False, {}

def test_case_header(test_name: str):
    """Hiển thị header cho test case"""
    print()
    print("=" * 60)
    info(f"🧪 Test Case: {test_name}")
    print("=" * 60)

def main():
    # Cấu hình users
    editor_email = "editor@gmail.com"
    editor_password = "123456"
    bob_email = "bob@gmail.com"
    
    print()
    info("Bắt đầu test các trường hợp gán role cho user")
    
    # ==========================================
    # Test Case 1: Admin gán super_admin cho chính mình
    # ==========================================
    test_case_header("1. Admin gán super_admin cho chính mình")
    
    # Login admin
    config = get_config()
    admin_token, admin_user = login(config["admin_email"], config["admin_password"])
    admin_user_id = admin_user.get("id")
    
    if not admin_user_id:
        error("Không thể lấy user_id của admin")
        sys.exit(1)
    
    info(f"Admin User ID: {admin_user_id}")
    
    # Lấy role_id của super_admin
    super_admin_role_id = get_role_id_by_name(admin_token, "super_admin")
    if not super_admin_role_id:
        error("Không tìm thấy role super_admin")
        sys.exit(1)
    
    info(f"Super Admin Role ID: {super_admin_role_id}")
    
    # Thử gán (mong đợi lỗi)
    test1_success, _ = assign_role_to_user(
        admin_token, admin_user_id, super_admin_role_id, expected_success=False
    )
    
    # ==========================================
    # Test Case 2: Admin gán super_admin cho bob
    # ==========================================
    test_case_header("2. Admin gán super_admin cho bob@gmail.com")
    
    # Lấy user_id của bob
    user_detail = get_user_detail(admin_token, bob_email)
    if not user_detail or "user" not in user_detail:
        error(f"Không thể lấy user_id của {bob_email}. Có thể user chưa tồn tại.")
        sys.exit(1)
    
    bob_user_id = user_detail["user"].get("id")
    if not bob_user_id:
        error(f"Không thể lấy user_id của {bob_email} từ response.")
        sys.exit(1)
    
    info(f"Bob User ID: {bob_user_id}")
    
    # Thử gán (mong đợi lỗi)
    test2_success, _ = assign_role_to_user(
        admin_token, bob_user_id, super_admin_role_id, expected_success=False
    )
    
    # ==========================================
    # Test Case 3: Admin gán admin cho bob
    # ==========================================
    test_case_header("3. Admin gán admin cho bob@gmail.com")
    
    # Lấy role_id của admin
    admin_role_id = get_role_id_by_name(admin_token, "admin")
    if not admin_role_id:
        error("Không tìm thấy role admin")
        sys.exit(1)
    
    info(f"Admin Role ID: {admin_role_id}")
    
    # Thử gán (mong đợi lỗi)
    test3_success, _ = assign_role_to_user(
        admin_token, bob_user_id, admin_role_id, expected_success=False
    )
    
    # ==========================================
    # Test Case 4: Admin gán editor cho bob
    # ==========================================
    test_case_header("4. Admin gán editor cho bob@gmail.com")
    
    # Lấy role_id của editor
    editor_role_id = get_role_id_by_name(admin_token, "editor")
    if not editor_role_id:
        error("Không tìm thấy role editor")
        sys.exit(1)
    
    info(f"Editor Role ID: {editor_role_id}")
    
    # Thử gán (mong đợi thành công)
    test4_success, _ = assign_role_to_user(
        admin_token, bob_user_id, editor_role_id, expected_success=True
    )
    
    # ==========================================
    # Test Case 5: Editor gán reader cho bob (lỗi authorization)
    # ==========================================
    test_case_header("5. Editor gán reader cho bob@gmail.com (lỗi authorization)")
    
    # Login editor
    editor_token, editor_user = login(editor_email, editor_password)
    
    # Lấy role_id của reader
    reader_role_id = get_role_id_by_name(editor_token, "reader")
    if not reader_role_id:
        error("Không tìm thấy role reader")
        sys.exit(1)
    
    info(f"Reader Role ID: {reader_role_id}")
    
    # Thử gán (mong đợi lỗi authorization)
    test5_success, _ = assign_role_to_user(
        editor_token, bob_user_id, reader_role_id, expected_success=False
    )
    
    # ==========================================
    # Tổng kết
    # ==========================================
    print()
    print("=" * 60)
    info("📊 Tổng kết kết quả test")
    print("=" * 60)
    
    total_tests = 5
    passed_tests = sum([
        test1_success,
        test2_success,
        test3_success,
        test4_success,
        test5_success
    ])
    
    results = [
        ("Test 1: Admin gán super_admin cho chính mình", test1_success),
        ("Test 2: Admin gán super_admin cho bob", test2_success),
        ("Test 3: Admin gán admin cho bob", test3_success),
        ("Test 4: Admin gán editor cho bob", test4_success),
        ("Test 5: Editor gán reader cho bob", test5_success),
    ]
    
    for test_name, result in results:
        if result:
            success(f"{test_name}: PASSED")
        else:
            error(f"{test_name}: FAILED")
    
    print()
    if passed_tests == total_tests:
        success(f"Tất cả {total_tests} test cases đã PASSED! 🎉")
        sys.exit(0)
    else:
        error(f"Có {total_tests - passed_tests} test cases FAILED")
        sys.exit(1)

if __name__ == "__main__":
    main()

