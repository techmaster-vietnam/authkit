#!/usr/bin/env python3
"""Script tự động login và tạo role mới"""
import json
import sys
import requests
from share import info, success, error, login, get_config, handle_error_response, get_base_url

# Ví dụ sử dụng:
# python3 create_role.py                    # Tạo role với ID=7, name="bar", is_system=False (mặc định)
# python3 create_role.py 10                # Tạo role với ID=10, name="bar", is_system=False
# python3 create_role.py 10 "admin"        # Tạo role với ID=10, name="admin", is_system=False
# python3 create_role.py 10 "user" true    # Tạo role với ID=10, name="user", is_system=True

def main():
    # Parse args
    role_id = int(sys.argv[1]) if len(sys.argv) > 1 else 7
    role_name = sys.argv[2] if len(sys.argv) > 2 else "bar"
    is_system = sys.argv[3].lower() not in ("false", "0") if len(sys.argv) > 3 else False
    
    # Login
    config = get_config()
    token, user = login(config["admin_email"], config["admin_password"])
    
    # Create role
    print()
    info("Đang tạo role mới...")
    info(f"  - ID: {role_id}")
    info(f"  - Name: {role_name}")
    info(f"  - Is System: {is_system}")
    
    resp = requests.post(
        f"{get_base_url()}/api/roles",
        json={"id": role_id, "name": role_name, "is_system": is_system},
        headers={"Authorization": f"Bearer {token}"}
    )
    
    print()
    info("Response từ server:")
    resp_data = resp.json()
    print(json.dumps(resp_data, indent=2, ensure_ascii=False))
    
    if resp.status_code >= 400 or "error" in resp_data:
        handle_error_response(resp_data, "tạo role")
        sys.exit(1)
    
    success("Tạo role thành công!")
    print()
    info("Thông tin role đã tạo:")
    print(json.dumps(resp_data.get("data", {}), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()

