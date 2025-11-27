#!/usr/bin/env python3
"""Script tự động login và xóa role"""
import json
import sys
import requests
from share import info, success, error, login, get_config, handle_error_response, get_base_url

def main():
    # Parse args
    role_id = int(sys.argv[1]) if len(sys.argv) > 1 else 6
    
    # Login
    config = get_config()
    token, user = login(config["admin_email"], config["admin_password"])
    
    # Delete role
    print()
    info("Đang xóa role...")
    info(f"  - Role ID: {role_id}")
    
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
        sys.exit(1)
    
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
            sys.exit(1)

if __name__ == "__main__":
    main()

