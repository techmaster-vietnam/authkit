#!/usr/bin/env python3
"""Module chứa các hàm dùng chung cho các script test"""
import json
import os
import sys
from typing import Dict, Tuple, Optional

try:
    import requests
except ImportError:
    print("❌ Cần cài đặt requests: pip install requests")
    sys.exit(1)

def get_config() -> Dict[str, str]:
    """Lấy cấu hình từ environment variables hoặc giá trị mặc định"""
    return {
        "base_url":"http://localhost:3000",
        "admin_email": "admin@gmail.com",
        "admin_password": "123456",
    }

# Biến toàn cục read-only cho base_url
_BASE_URL: str = get_config()["base_url"]

# Colors
RED = '\033[0;31m'
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
RESET = '\033[0m'

def info(msg: str): 
    """Hiển thị thông báo thông tin"""
    print(f"{YELLOW}ℹ️  {msg}{RESET}")

def success(msg: str): 
    """Hiển thị thông báo thành công"""
    print(f"{GREEN}✅ {msg}{RESET}")

def error(msg: str): 
    """Hiển thị thông báo lỗi"""
    print(f"{RED}❌ {msg}{RESET}")



def get_base_url() -> str:
    """Lấy base_url (read-only)"""
    return _BASE_URL

def login(email: str, password: str) -> Tuple[str, Dict]:
    """
    Thực hiện login và trả về token cùng thông tin user
    
    Args:
        email: Email để login (bắt buộc)
        password: Password để login (bắt buộc)
    
    Returns:
        Tuple (token, user_info)
    
    Raises:
        SystemExit: Nếu login thất bại
    """
    base_url = _BASE_URL
    
    info(f"Đang đăng nhập với email: {email}...")
    resp = requests.post(
        f"{base_url}/api/auth/login", 
        json={"email": email, "password": password}
    )
    resp.raise_for_status()
    data = resp.json()
    
    if "error" in data:
        error("Lỗi đăng nhập:")
        print(json.dumps(data, indent=2, ensure_ascii=False))
        sys.exit(1)
    
    if "data" not in data:
        error("Response không hợp lệ:")
        print(json.dumps(data, indent=2, ensure_ascii=False))
        sys.exit(1)
    
    token = data.get("data", {}).get("token")
    if not token:
        error("Không thể lấy token từ response:")
        print(json.dumps(data, indent=2, ensure_ascii=False))
        sys.exit(1)
    
    user = data.get("data", {}).get("user", {})
    success("Đăng nhập thành công!")
    info(f"Token: {token[:50]}...")
    info(f"User ID: {user.get('id', 'N/A')}, Email: {user.get('email', 'N/A')}")
    
    return token, user

def handle_error_response(resp_data: Dict, operation: str = "thao tác") -> None:
    """
    Xử lý và hiển thị lỗi từ response
    
    Args:
        resp_data: Dictionary chứa response từ server
        operation: Tên thao tác đang thực hiện (để hiển thị trong thông báo lỗi)
    """
    error(f"Lỗi khi {operation}")
    
    error_type = resp_data.get("type", "UNKNOWN")
    error_value = resp_data.get("error", "")
    
    # Xử lý error có thể là string hoặc object
    if isinstance(error_value, dict):
        error_msg = error_value.get("message", str(error_value))
    else:
        error_msg = str(error_value)
    
    error(f"Loại lỗi: {error_type}")
    error(f"Chi tiết: {error_msg}")
    
    # Hiển thị thêm thông tin nếu có
    if "data" in resp_data:
        info("Thông tin thêm:")
        print(json.dumps(resp_data.get("data"), indent=2, ensure_ascii=False))

def get_role_id_by_name(token: str, role_name: str) -> Optional[int]:
    """
    Lấy role_id từ role name
    
    Args:
        token: JWT token để xác thực
        role_name: Tên role cần tìm
    
    Returns:
        role_id hoặc None nếu không tìm thấy
    """
    try:
        resp = requests.get(
            f"{_BASE_URL}/api/roles",
            headers={"Authorization": f"Bearer {token}"}
        )
        resp.raise_for_status()
        data = resp.json()
        
        if "data" in data:
            for role in data["data"]:
                if role.get("name") == role_name:
                    return role.get("id")
        return None
    except Exception as e:
        error(f"Lỗi khi lấy role_id cho {role_name}: {str(e)}")
        return None

def get_user_detail(token: str, identifier: str, verbose: bool = True) -> Optional[Dict]:
    """
    Lấy thông tin chi tiết người dùng theo ID hoặc email
    
    Args:
        token: JWT token để xác thực
        identifier: ID hoặc email của user cần lấy thông tin
        verbose: Nếu True, in ra thông tin chi tiết. Mặc định là True
    
    Returns:
        Dictionary chứa thông tin user và roles, hoặc None nếu thất bại
    """
    # Gọi API để lấy user detail
    if verbose:
        info(f"Đang lấy thông tin chi tiết cho: {identifier}...")
    try:
        resp = requests.get(
            f"{_BASE_URL}/api/users/detail",
            params={"identifier": identifier},
            headers={"Authorization": f"Bearer {token}"}
        )
        
        # Kiểm tra status code
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, "lấy thông tin chi tiết user")
            except:
                error(f"Response: {resp.text}")
            return None
        
        data = resp.json()
        
        # Kiểm tra response có lỗi không
        if "error" in data:
            handle_error_response(data, "lấy thông tin chi tiết user")
            return None
        
        # Kiểm tra có data không
        if "data" not in data:
            error("Response không hợp lệ:")
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return None
        
        if verbose:
            success("Lấy thông tin chi tiết user thành công!")
        user_detail = data.get("data", {})
        
        # In ra thông tin user (chỉ khi verbose=True)
        if verbose:
            user = user_detail.get("user", {})
            roles = user_detail.get("roles", [])
            
            info(f"User ID: {user.get('id', 'N/A')}")
            info(f"Email: {user.get('email', 'N/A')}")
            info(f"Full Name: {user.get('full_name', 'N/A')}")
            info(f"Is Active: {user.get('is_active', 'N/A')}")
            info(f"Số lượng roles: {len(roles)}")
            
            if roles:
                info("Danh sách roles:")
                for role in roles:
                    print(f"  - Role ID: {role.get('role_id')}, Role Name: {role.get('role_name')}")
        
        return user_detail
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi khi gọi API: {str(e)}")
        return None
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return None

def get_user_roles(token: str, identifier: str) -> Optional[list]:
    """
    Lấy danh sách roles của user theo ID hoặc email
    
    Args:
        token: JWT token để xác thực
        identifier: ID hoặc email của user cần lấy roles
    
    Returns:
        List các roles dưới dạng [[role_id, role_name], ...], hoặc None nếu thất bại
    """
    # Gọi API để lấy user detail
    info(f"Đang lấy danh sách roles cho: {identifier}...")
    try:
        resp = requests.get(
            f"{_BASE_URL}/api/users/detail",
            params={"identifier": identifier},
            headers={"Authorization": f"Bearer {token}"}
        )
        
        # Kiểm tra status code
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, "lấy danh sách roles của user")
            except:
                error(f"Response: {resp.text}")
            return None
        
        data = resp.json()
        
        # Kiểm tra response có lỗi không
        if "error" in data:
            handle_error_response(data, "lấy danh sách roles của user")
            return None
        
        # Kiểm tra có data không
        if "data" not in data:
            error("Response không hợp lệ:")
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return None
        
        user_detail = data.get("data", {})
        roles = user_detail.get("roles", [])
        
        # Lọc và format dữ liệu roles thành [role_id, role_name]
        result = []
        for role in roles:
            role_id = role.get('role_id')
            role_name = role.get('role_name')
            if role_id is not None and role_name:
                result.append([role_id, role_name])
        
        success(f"Lấy danh sách roles thành công! Tìm thấy {len(result)} roles")
        return result
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi khi gọi API: {str(e)}")
        return None
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return None

def create_role(token: str, role_id: int, role_name: str, is_system: bool = False) -> bool:
    """
    Tạo role mới
    
    Args:
        token: JWT token để xác thực
        role_id: ID của role
        role_name: Tên role
        is_system: Có phải system role không
    
    Returns:
        True nếu thành công, False nếu thất bại
    """
    try:
        info("Đang tạo role mới...")
        info(f"  - ID: {role_id}")
        info(f"  - Name: {role_name}")
        info(f"  - Is System: {is_system}")
        
        resp = requests.post(
            f"{_BASE_URL}/api/roles",
            json={"id": role_id, "name": role_name, "is_system": is_system},
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "tạo role")
            return False
        
        return True
        
    except Exception as e:
        error(f"Lỗi khi tạo role: {str(e)}")
        return False

def update_user_roles(token: str, user_id: str, role_names: list) -> Tuple[bool, Optional[Dict]]:
    """
    Cập nhật danh sách roles cho user
    
    Args:
        token: JWT token để xác thực
        user_id: ID của user cần cập nhật roles
        role_names: Danh sách tên roles (ví dụ: ["author", "reader", "tiger"])
    
    Returns:
        Tuple (success, response_data)
    """
    try:
        info(f"Đang cập nhật roles cho user {user_id}...")
        info(f"  - Danh sách roles: {role_names}")
        
        resp = requests.put(
            f"{_BASE_URL}/api/users/{user_id}/roles",
            json={"roles": role_names},
            headers={"Authorization": f"Bearer {token}"}
        )
        
        resp_data = resp.json()
        
        print()
        info("Response từ server:")
        print(json.dumps(resp_data, indent=2, ensure_ascii=False))
        
        if resp.status_code >= 400 or "error" in resp_data:
            handle_error_response(resp_data, "cập nhật roles cho user")
            return False, resp_data
        
        success("Cập nhật roles thành công!")
        return True, resp_data
        
    except Exception as e:
        error(f"Lỗi khi cập nhật roles: {str(e)}")
        return False, None

def display_user_roles(user_detail: Optional[Dict], title: str = "Danh sách roles") -> list:
    """
    Hiển thị danh sách roles của user và trả về danh sách role names
    
    Args:
        user_detail: Dictionary chứa thông tin user detail từ get_user_detail
        title: Tiêu đề để hiển thị
    
    Returns:
        List các role names (ví dụ: ["author", "reader", "tiger"])
    """
    print()
    print("=" * 60)
    info(f"📋 {title}")
    print("=" * 60)
    
    if not user_detail or "roles" not in user_detail:
        info("Không có role nào")
        return []
    
    roles = user_detail.get("roles", [])
    role_names = []
    
    if roles:
        for idx, role in enumerate(roles, 1):
            role_id = role.get('role_id', 'N/A')
            role_name = role.get('role_name', 'N/A')
            print(f"{idx}. Role ID: {role_id}, Role Name: {role_name}")
            if role_name != 'N/A':
                role_names.append(role_name)
    else:
        info("Không có role nào")
    
    print()
    return role_names
