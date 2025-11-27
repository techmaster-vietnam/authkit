#!/usr/bin/env python3
"""Script lấy thông tin chi tiết của các user"""
import sys
from share import info, success, error, get_user_detail, get_config, login

def main():
    """Hàm main để lấy thông tin chi tiết của các user"""
    # Danh sách các user cần lấy thông tin (có thể là email hoặc ID)
    user_identifiers = [
        "editor@gmail.com",
        "author1@gmail.com",
        "bob@gmail.com",
        "JOQkg6Ao4S7V"
    ]
    
    print()
    info("Bắt đầu script lấy thông tin chi tiết các user")
    print()
    
    # Login một lần với admin account để lấy token
    info("Đang đăng nhập với admin account...")
    try:
        token, _ = login("admin@gmail.com", "123456")
    except SystemExit:
        error("Không thể đăng nhập với admin account")
        sys.exit(1)
    except Exception as e:
        error(f"Lỗi khi đăng nhập: {str(e)}")
        sys.exit(1)
    
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
    success("Script hoàn thành!")
    print("=" * 80)

if __name__ == "__main__":
    main()

