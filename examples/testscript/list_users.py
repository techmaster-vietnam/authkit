#!/usr/bin/env python3
"""Script test list users với pagination, sort và filter"""
import json
import os
import sys
from typing import Dict, List, Optional, Tuple

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

from share import (
    info, success, error, get_base_url, print_section,
    login_account, delete_user, handle_error_response
)

# Định nghĩa cấu trúc user
UserData = Dict[str, str]

def register_user(user_data: UserData) -> Tuple[bool, Optional[Dict], Optional[str]]:
    """
    Đăng ký user mới (không in thông báo nếu thành công)
    
    Args:
        user_data: Dictionary chứa thông tin user (email, password, full_name, mobile, address)
    
    Returns:
        Tuple (success, user_info, error_message)
        - success: True nếu đăng ký thành công, False nếu lỗi
        - user_info: Thông tin user nếu thành công, None nếu lỗi
        - error_message: Thông báo lỗi nếu có, None nếu thành công
    """
    base_url = get_base_url()
    
    # Chuẩn bị request body
    request_body = {
        "email": user_data.get("email", ""),
        "password": user_data.get("password", ""),
        "full_name": user_data.get("full_name", ""),
    }
    
    # Thêm các trường custom nếu có
    if "mobile" in user_data:
        request_body["mobile"] = user_data["mobile"]
    if "address" in user_data:
        request_body["address"] = user_data["address"]
    
    try:
        resp = requests.post(
            f"{base_url}/api/auth/register",
            json=request_body,
            timeout=10
        )
        
        # Parse response
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, None, f"Response không phải JSON. Status: {resp.status_code}"
        
        # Kiểm tra lỗi
        if resp.status_code != 201:
            error_msg = "Lỗi không xác định"
            
            # Thử lấy từ "error" object (nếu là dict)
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
            elif isinstance(error_obj, str):
                error_msg = error_obj
            
            # Thử lấy từ top level "message" (format của goerrorkit)
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            
            # Thử lấy chi tiết validation từ top level "data"
            error_details = {}
            if "data" in resp_data and isinstance(resp_data.get("data"), dict):
                error_details = resp_data.get("data", {})
            
            # Format error message
            if error_details:
                error_msg += f" | Chi tiết: {json.dumps(error_details, ensure_ascii=False)}"
            
            return False, None, error_msg
        
        # Kiểm tra response có data không
        if "data" not in resp_data:
            return False, None, "Response không chứa data"
        
        user_info = resp_data.get("data", {})
        return True, user_info, None
        
    except requests.exceptions.RequestException as e:
        return False, None, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, None, f"Lỗi không xác định: {str(e)}"

def list_users(token: str, page: int = 1, page_size: int = 10, 
               email_filter: str = None, full_name_filter: str = None, 
               address_filter: str = None, sort_by: str = None, 
               order: str = "asc") -> Optional[Dict]:
    """
    Lấy danh sách users với pagination, filter và sort
    
    Args:
        token: JWT token để xác thực
        page: Số trang (bắt đầu từ 1)
        page_size: Số lượng items mỗi trang
        email_filter: Filter email chứa text
        full_name_filter: Filter full_name chứa text
        address_filter: Filter address chứa text
        sort_by: Trường để sort (email, full_name, address)
        order: Thứ tự sort (asc hoặc desc)
    
    Returns:
        Dictionary chứa response từ API hoặc None nếu thất bại
    """
    try:
        base_url = get_base_url()
        
        # Xây dựng query parameters
        params = {
            "page": page,
            "page_size": page_size,
        }
        
        # Thêm filters
        if email_filter:
            params["email"] = email_filter
        if full_name_filter:
            params["full_name"] = full_name_filter
        if address_filter:
            params["address"] = address_filter
        
        # Thêm sort params (giả định API hỗ trợ)
        if sort_by:
            params["sort_by"] = sort_by
            params["order"] = order
        
        resp = requests.get(
            f"{base_url}/api/user",
            params=params,
            headers={"Authorization": f"Bearer {token}"}
        )
        
        # Kiểm tra status code
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, "lấy danh sách users")
            except:
                error(f"Response: {resp.text}")
            return None
        
        data = resp.json()
        
        # Kiểm tra response có lỗi không
        if "error" in data:
            handle_error_response(data, "lấy danh sách users")
            return None
        
        return data
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi khi gọi API: {str(e)}")
        return None
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return None

def print_users_list(response_data: Dict, title: str = "Danh sách users"):
    """
    In danh sách users từ response
    
    Args:
        response_data: Dictionary chứa response từ API
        title: Tiêu đề để hiển thị
    """
    print()
    print("=" * 80)
    info(f"📋 {title}")
    print("=" * 80)
    
    if not response_data or "data" not in response_data:
        error("Response không hợp lệ")
        print()
        return
    
    # Lấy data object từ response
    data_obj = response_data.get("data", {})
    
    # Lấy danh sách users từ data object
    users = data_obj.get("users", [])
    
    # Kiểm tra users có phải là list không
    if not isinstance(users, list):
        error(f"Response không hợp lệ: 'users' không phải là list. Type: {type(users)}")
        info(f"Response data: {json.dumps(response_data, indent=2, ensure_ascii=False)}")
        print()
        return
    
    # Lấy thông tin pagination nếu có
    pagination_enabled = data_obj.get("pagination_enabled", False)
    total = data_obj.get("total", 0)
    page = data_obj.get("page")
    page_size = data_obj.get("page_size")
    total_pages = data_obj.get("total_pages")
    
    if pagination_enabled:
        if page is not None and total_pages is not None:
            info(f"Trang: {page}/{total_pages}")
        if total is not None:
            info(f"Tổng số: {total} users")
        if page_size is not None:
            info(f"Số lượng trên trang này: {len(users)}/{page_size} users")
    else:
        info(f"Tổng số: {len(users)} users (không phân trang)")
    
    if not users:
        info("Không có user nào")
        print()
        return
    
    print()
    print("-" * 80)
    for idx, user in enumerate(users, 1):
        # Kiểm tra user có phải là dictionary không
        if not isinstance(user, dict):
            error(f"User không phải là dictionary. Type: {type(user)}, Value: {user}")
            continue
        
        user_id = user.get("id", "N/A")
        email = user.get("email", "N/A")
        full_name = user.get("full_name", "N/A")
        mobile = user.get("mobile", "N/A")
        address = user.get("address", "N/A")
        
        print(f"{idx}. ID: {user_id}")
        print(f"   Email: {email}")
        print(f"   Full Name: {full_name}")
        print(f"   Mobile: {mobile}")
        print(f"   Address: {address}")
        print("-" * 80)
    
    print()

def main():
    """Hàm main để test list users"""
    
    print_section("BẮT ĐẦU SCRIPT TEST LIST USERS")
    
    # Bước 1: Đọc file users.json
    print_section("BƯỚC 1: ĐỌC FILE USERS.JSON")
    
    script_dir = os.path.dirname(os.path.abspath(__file__))
    users_file = os.path.join(script_dir, "users.json")
    
    if not os.path.exists(users_file):
        error(f"Không tìm thấy file: {users_file}")
        sys.exit(1)
    
    try:
        with open(users_file, 'r', encoding='utf-8') as f:
            users_data = json.load(f)
        success(f"Đọc file thành công! Tổng số users: {len(users_data)}")
    except Exception as e:
        error(f"Lỗi khi đọc file: {str(e)}")
        sys.exit(1)
    
    print()
    
    # Bước 2: Register users
    print_section("BƯỚC 2: ĐĂNG KÝ USERS")
    
    registered_user_ids: List[str] = []
    success_count = 0
    error_count = 0
    
    for idx, user_data in enumerate(users_data, 1):
        # Thêm password mặc định
        user_data["password"] = "123456"
        
        register_success, user_info, error_msg = register_user(user_data)
        
        if register_success:
            success_count += 1
            if user_info and user_info.get('id'):
                registered_user_ids.append(user_info.get('id'))
        else:
            error_count += 1
            error(f"[{idx}/{len(users_data)}] {user_data.get('email', 'N/A')}: {error_msg}")
    
    print()
    info(f"Tổng kết đăng ký: {success_count} thành công, {error_count} thất bại")
    print()
    
    if not registered_user_ids:
        error("Không có user nào được đăng ký thành công. Không thể tiếp tục test.")
        sys.exit(1)
    
    # Bước 3: Login với admin/super_admin để test list users
    print_section("BƯỚC 3: ĐĂNG NHẬP ADMIN")
    
    login_success, admin_token, login_error = login_account("super_admin")
    
    if not login_success:
        error(f"Không thể đăng nhập: {login_error}")
        sys.exit(1)
    
    success("Đăng nhập thành công!")
    print()
    
    # Bước 4: Test các kịch bản list users
    print_section("BƯỚC 4: TEST CÁC KỊCH BẢN LIST USERS")
    
    # Test case A: Phân trang, sort theo email A-Z, in ra trang đầu tiên
    print()
    print("=" * 80)
    info("TEST CASE A: Phân trang, sort theo email A-Z, trang đầu tiên")
    print("=" * 80)
    response_a = list_users(admin_token, page=1, page_size=10, sort_by="email", order="asc")
    if response_a:
        print_users_list(response_a, "Kết quả Test Case A")
    
    # Test case B: Phân trang, sort theo email Z-A, in ra trang đầu tiên
    print()
    print("=" * 80)
    info("TEST CASE B: Phân trang, sort theo email Z-A, trang đầu tiên")
    print("=" * 80)
    response_b = list_users(admin_token, page=1, page_size=10, sort_by="email", order="desc")
    if response_b:
        print_users_list(response_b, "Kết quả Test Case B")
    
    # Test case C: Phân trang, sort theo full_name A-Z, in ra trang đầu tiên
    print()
    print("=" * 80)
    info("TEST CASE C: Phân trang, sort theo full_name A-Z, trang đầu tiên")
    print("=" * 80)
    response_c = list_users(admin_token, page=1, page_size=10, sort_by="full_name", order="asc")
    if response_c:
        print_users_list(response_c, "Kết quả Test Case C")
    
    # Test case D: Phân trang, sort theo full_name Z-A, in ra trang đầu tiên
    print()
    print("=" * 80)
    info("TEST CASE D: Phân trang, sort theo full_name Z-A, trang đầu tiên")
    print("=" * 80)
    response_d = list_users(admin_token, page=1, page_size=10, sort_by="full_name", order="desc")
    if response_d:
        print_users_list(response_d, "Kết quả Test Case D")
    
    # Test case E: Lọc ra email chứa "micro", sort theo email A-Z
    print()
    print("=" * 80)
    info("TEST CASE E: Lọc email chứa 'micro', sort theo email A-Z")
    print("=" * 80)
    response_e = list_users(admin_token, page=1, page_size=10, email_filter="micro", sort_by="email", order="asc")
    if response_e:
        print_users_list(response_e, "Kết quả Test Case E")
    
    # Test case F: Lọc ra email chứa "uni", sort theo email Z-A
    print()
    print("=" * 80)
    info("TEST CASE F: Lọc email chứa 'uni', sort theo email Z-A")
    print("=" * 80)
    response_f = list_users(admin_token, page=1, page_size=10, email_filter="uni", sort_by="email", order="desc")
    if response_f:
        print_users_list(response_f, "Kết quả Test Case F")
    
    # Test case G: Lọc ra full_name chứa "son", sort theo full_name A-Z
    print()
    print("=" * 80)
    info("TEST CASE G: Lọc full_name chứa 'son', sort theo full_name A-Z")
    print("=" * 80)
    response_g = list_users(admin_token, page=1, page_size=10, full_name_filter="son", sort_by="full_name", order="asc")
    if response_g:
        print_users_list(response_g, "Kết quả Test Case G")
    
    # Test case H: Lọc ra address chứa "way", sort theo address A-Z
    print()
    print("=" * 80)
    info("TEST CASE H: Lọc address chứa 'way', sort theo address A-Z")
    print("=" * 80)
    response_h = list_users(admin_token, page=1, page_size=10, address_filter="way", sort_by="address", order="asc")
    if response_h:
        print_users_list(response_h, "Kết quả Test Case H")
    
    # Bước 5: Xác nhận để tiếp tục
    print_section("BƯỚC 5: XÁC NHẬN ĐỂ TIẾP TỤC")
    
    print()
    print("Nhấn bất kỳ phím nào để tiếp tục xóa các users đã đăng ký...")
    try:
        input()
    except KeyboardInterrupt:
        print()
        info("Đã hủy bởi người dùng.")
        sys.exit(0)
    except EOFError:
        print()
        info("Đã hủy bởi người dùng.")
        sys.exit(0)
    
    # Bước 6: Xóa toàn bộ users đã đăng ký
    print_section("BƯỚC 6: XÓA TOÀN BỘ USERS ĐÃ ĐĂNG KÝ")
    
    info(f"Tổng số user sẽ bị xóa: {len(registered_user_ids)}")
    print()
    
    delete_success_count = 0
    delete_fail_count = 0
    
    for idx, user_id in enumerate(registered_user_ids, 1):
        info(f"[{idx}/{len(registered_user_ids)}] Đang xóa user ID: {user_id}...")
        delete_success, delete_error = delete_user(admin_token, user_id)
        if delete_success:
            delete_success_count += 1
        else:
            delete_fail_count += 1
            error(f"Xóa user ID {user_id} thất bại: {delete_error}")
        print()
    
    # Báo cáo kết quả xóa
    print()
    print_section("KẾT QUẢ XÓA USERS")
    print(f"   ✅ Xóa thành công: {delete_success_count}")
    print(f"   ❌ Xóa thất bại: {delete_fail_count}")
    print()

if __name__ == "__main__":
    main()

