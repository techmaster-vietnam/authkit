#!/usr/bin/env python3
"""Script để cập nhật rule"""
import sys
from share import (
    get_config,
    login_with_error_handling,
    update_rule,
    get_rule_by_id,
    print_rule_detail,
    get_role_names_map,
    print_section,
    success,
    error,
    info
)

def main():
    """Hàm main"""
    config = get_config()
    
    print_section("CẬP NHẬT RULE")
    
    # 1. Login với admin@gmail.com
    print_section("Bước 1: Đăng nhập với admin")
    admin_email = config.get("admin_email", "admin@gmail.com")
    admin_password = config.get("admin_password", "123456")
    
    try:
        token = login_with_error_handling(admin_email, admin_password, "admin")
    except SystemExit:
        error("Không thể đăng nhập với admin account")
        sys.exit(1)
    
    # Lấy role names map để hiển thị
    role_names_map = get_role_names_map(token)
    
    rule_id = "GET|/api/bar"
    
    # 2. Cập nhật rule lần đầu
    print_section("Bước 2: Cập nhật rule lần đầu")
    success_flag, updated_rule = update_rule(
        token=token,
        rule_id=rule_id,
        rule_type="ALLOW",
        roles=["author"],
        description="Ho ho ha ha he he",
        verbose=True
    )
    
    if not success_flag:
        error("Không thể cập nhật rule lần đầu")
        sys.exit(1)
    
    # 3. Hiển thị thông tin rule sau khi cập nhật
    print_section("Bước 3: Hiển thị thông tin rule sau khi cập nhật")
    rule = get_rule_by_id(token, rule_id, verbose=True)
    if rule:
        print_rule_detail(token, rule, "Thông tin rule sau khi cập nhật lần đầu", role_names_map)
    else:
        error("Không thể lấy thông tin rule")
        sys.exit(1)
    
    # 4. Cập nhật lại rule lần thứ hai
    print_section("Bước 4: Cập nhật rule lần thứ hai")
    success_flag, updated_rule = update_rule(
        token=token,
        rule_id=rule_id,
        rule_type="FORBID",
        roles=["reader", "editor", "admin"],
        description="Bar dùng Override() để ghi đè rule code từ database",
        verbose=True
    )
    
    if not success_flag:
        error("Không thể cập nhật rule lần thứ hai")
        sys.exit(1)
    
    # Hiển thị thông tin rule sau khi cập nhật lần thứ hai
    print_section("Bước 5: Hiển thị thông tin rule sau khi cập nhật lần thứ hai")
    rule = get_rule_by_id(token, rule_id, verbose=True)
    if rule:
        print_rule_detail(token, rule, "Thông tin rule sau khi cập nhật lần thứ hai", role_names_map)
    else:
        error("Không thể lấy thông tin rule")
        sys.exit(1)
    
    print_section("HOÀN THÀNH")
    success("Đã cập nhật rule thành công!")

if __name__ == "__main__":
    main()

