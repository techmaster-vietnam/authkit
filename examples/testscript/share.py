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
        "super_admin_email": "superadmin@gmail.com",
        "super_admin_password": "123456",
    }

# Biến toàn cục read-only cho base_url
_BASE_URL: str = get_config()["base_url"]

# Colors
RED = '\033[0;31m'
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
BLUE = '\033[0;34m'
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
    Lấy thông tin chi tiết người dùng theo ID, email hoặc mobile
    
    Args:
        token: JWT token để xác thực
        identifier: ID, email hoặc mobile của user cần lấy thông tin
        verbose: Nếu True, in ra thông tin chi tiết. Mặc định là True
    
    Returns:
        Dictionary chứa thông tin user và roles, hoặc None nếu thất bại
    """
    # Gọi API để lấy user detail
    if verbose:
        info(f"Đang lấy thông tin chi tiết cho: {identifier}...")
    try:
        # URL encode identifier để đảm bảo an toàn khi truyền trong URL path
        from urllib.parse import quote
        encoded_identifier = quote(identifier, safe='')
        resp = requests.get(
            f"{_BASE_URL}/api/auth/profile/{encoded_identifier}",
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

def print_section(title: str):
    """
    In tiêu đề section với format đẹp
    
    Args:
        title: Tiêu đề section cần in
    """
    print()
    print("=" * 80)
    print(f"{BLUE}{title}{RESET}")
    print("=" * 80)
    print()


def login_with_error_handling(email: str, password: str, account_name: str = None) -> str:
    """
    Login với error handling tự động
    
    Args:
        email: Email để đăng nhập
        password: Mật khẩu
        account_name: Tên account để hiển thị trong error message (mặc định là email)
    
    Returns:
        Token nếu thành công
    
    Raises:
        SystemExit: Nếu login thất bại
    """
    if account_name is None:
        account_name = email
    
    try:
        token, _ = login(email, password)
        return token
    except SystemExit:
        error(f"Không thể đăng nhập với {account_name} account")
        sys.exit(1)
    except Exception as e:
        error(f"Lỗi khi đăng nhập: {str(e)}")
        sys.exit(1)

def login_account(account_type: str = "super_admin") -> Tuple[bool, Optional[str], Optional[str]]:
    """
    Login với account từ config (super_admin hoặc admin) mà không exit chương trình
    Hàm dùng chung để login với các account đặc biệt từ config
    
    Args:
        account_type: Loại account ("super_admin" hoặc "admin")
    
    Returns:
        Tuple (success, token, error_message)
        - success: True nếu login thành công, False nếu lỗi
        - token: JWT token nếu thành công, None nếu lỗi
        - error_message: Thông báo lỗi nếu có, None nếu thành công
    """
    config = get_config()
    
    if account_type == "super_admin":
        email_key = "super_admin_email"
        password_key = "super_admin_password"
        account_name = "super_admin"
        default_email = "superadmin@gmail.com"
    elif account_type == "admin":
        email_key = "admin_email"
        password_key = "admin_password"
        account_name = "admin"
        default_email = "admin@gmail.com"
    else:
        return False, None, f"Account type không hợp lệ: {account_type}. Chỉ hỗ trợ 'super_admin' hoặc 'admin'"
    
    email = config.get(email_key, default_email)
    password = config.get(password_key, "123456")
    
    try:
        token, _ = login(email, password)
        return True, token, None
    except SystemExit:
        return False, None, f"Không thể đăng nhập với {account_name} ({email}). Vui lòng kiểm tra password."
    except Exception as e:
        return False, None, f"Lỗi không mong đợi: {str(e)}"


def get_profile(token: str, verbose: bool = True) -> Optional[Dict]:
    """
    Lấy thông tin profile của chính mình
    
    Args:
        token: JWT token để xác thực
        verbose: Nếu True, in ra thông tin chi tiết. Mặc định là True
    
    Returns:
        Dictionary chứa thông tin user, hoặc None nếu thất bại
    """
    if verbose:
        info("Đang lấy thông tin profile của chính mình...")
    
    try:
        resp = requests.get(
            f"{_BASE_URL}/api/auth/profile",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        # Kiểm tra status code
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, "lấy thông tin profile")
            except:
                error(f"Response: {resp.text}")
            return None
        
        data = resp.json()
        
        # Kiểm tra response có lỗi không
        if "error" in data:
            handle_error_response(data, "lấy thông tin profile")
            return None
        
        # Kiểm tra có data không
        if "data" not in data:
            error("Response không hợp lệ:")
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return None
        
        if verbose:
            success("Lấy thông tin profile thành công!")
        
        user = data.get("data", {})
        
        # In ra thông tin user (chỉ khi verbose=True)
        if verbose:
            info(f"User ID: {user.get('id', 'N/A')}")
            info(f"Email: {user.get('email', 'N/A')}")
            info(f"Full Name: {user.get('full_name', 'N/A')}")
            info(f"Mobile: {user.get('mobile', 'N/A')}")
            info(f"Address: {user.get('address', 'N/A')}")
            info(f"Is Active: {user.get('is_active', 'N/A')}")
        
        return user
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi khi gọi API: {str(e)}")
        return None
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return None


def get_profile_by_identifier(token: str, identifier: str, verbose: bool = True) -> Optional[Dict]:
    """
    Lấy thông tin profile theo identifier (id, email, hoặc mobile)
    Chỉ dành cho admin và super_admin
    Sử dụng hàm get_user_detail để tận dụng code chung
    
    Args:
        token: JWT token để xác thực
        identifier: ID, email, hoặc mobile của user cần lấy thông tin
        verbose: Nếu True, in ra thông tin chi tiết. Mặc định là True
    
    Returns:
        Dictionary chứa thông tin user, hoặc None nếu thất bại
    """
    user_detail = get_user_detail(token, identifier, verbose)
    
    if user_detail:
        # Trả về user object từ user_detail để tương thích
        return user_detail.get("user", {})
    
    return None


def get_user_roles(token: str, identifier: str) -> Optional[list]:
    """
    Lấy danh sách roles của user theo ID, email hoặc mobile
    
    Args:
        token: JWT token để xác thực
        identifier: ID, email hoặc mobile của user cần lấy roles
    
    Returns:
        List các roles dưới dạng [[role_id, role_name], ...], hoặc None nếu thất bại
    """
    # Gọi API để lấy user detail
    info(f"Đang lấy danh sách roles cho: {identifier}...")
    try:
        # URL encode identifier để đảm bảo an toàn khi truyền trong URL path
        from urllib.parse import quote
        encoded_identifier = quote(identifier, safe='')
        resp = requests.get(
            f"{_BASE_URL}/api/auth/profile/{encoded_identifier}",
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

def filter_rules(token: str, method: str = None, path: str = None, type_param: str = None, fixed: bool = None, verbose: bool = True) -> Optional[list]:
    """
    Lọc và lấy danh sách rules theo các tiêu chí
    
    Args:
        token: JWT token để xác thực
        method: Method để lọc (GET, POST, PUT, DELETE) - tùy chọn
        path: Chuỗi để tìm trong path (LIKE search) - tùy chọn
        type_param: Type để lọc (PUBLIC, ALLOW, FORBID) - tùy chọn
        fixed: Fixed để lọc (True hoặc False) - tùy chọn
        verbose: Nếu True, in ra thông tin chi tiết. Mặc định là True
    
    Returns:
        List các rules hoặc None nếu thất bại
    """
    try:
        if verbose:
            info("Đang lấy danh sách rules...")
            if method:
                info(f"  - Method: {method}")
            if path:
                info(f"  - Path chứa: {path}")
            if type_param:
                info(f"  - Type: {type_param}")
            if fixed is not None:
                info(f"  - Fixed: {fixed}")
        
        # Xây dựng query parameters
        params = {}
        if method:
            params["method"] = method
        if path:
            params["path"] = path
        if type_param:
            params["type"] = type_param
        if fixed is not None:
            params["fixed"] = "true" if fixed else "false"
        
        resp = requests.get(
            f"{_BASE_URL}/api/rules",
            params=params,
            headers={"Authorization": f"Bearer {token}"}
        )
        
        # Kiểm tra status code
        if resp.status_code != 200:
            error(f"Request thất bại với status code: {resp.status_code}")
            try:
                error_data = resp.json()
                handle_error_response(error_data, "lấy danh sách rules")
            except:
                error(f"Response: {resp.text}")
            return None
        
        data = resp.json()
        
        # Kiểm tra response có lỗi không
        if "error" in data:
            handle_error_response(data, "lấy danh sách rules")
            return None
        
        # Kiểm tra có data không
        if "data" not in data:
            error("Response không hợp lệ:")
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return None
        
        rules = data.get("data", [])
        if verbose:
            success(f"Lấy danh sách rules thành công! Tìm thấy {len(rules)} rules")
        
        return rules
        
    except requests.exceptions.RequestException as e:
        error(f"Lỗi khi gọi API: {str(e)}")
        return None
    except Exception as e:
        error(f"Lỗi không mong đợi: {str(e)}")
        return None

def get_role_names_map(token: str) -> Dict[int, str]:
    """
    Lấy map role_id -> role_name từ API
    
    Args:
        token: JWT token để xác thực
    
    Returns:
        Dictionary mapping role_id -> role_name
    """
    try:
        resp = requests.get(
            f"{_BASE_URL}/api/roles",
            headers={"Authorization": f"Bearer {token}"}
        )
        resp.raise_for_status()
        data = resp.json()
        
        role_map = {}
        if "data" in data:
            for role in data["data"]:
                role_id = role.get("id")
                role_name = role.get("name")
                if role_id is not None and role_name:
                    role_map[role_id] = role_name
        return role_map
    except Exception as e:
        # Nếu không lấy được, trả về dict rỗng
        return {}

def print_rules_list(token: str, rules: Optional[list], title: str = "Danh sách rules", role_names_map: Dict[int, str] = None) -> None:
    """
    Hiển thị danh sách rules theo format: ID  , TYPE("role1", "role2") , fixed, service_name
    
    Args:
        token: JWT token để lấy role names (tùy chọn, chỉ dùng nếu role_names_map không được truyền)
        rules: List các rules từ filter_rules
        title: Tiêu đề để hiển thị
        role_names_map: Map role_id -> role_name để tái sử dụng (tùy chọn, nếu không có sẽ gọi API)
    """
    print()
    print("=" * 60)
    info(f"📋 {title}")
    print("=" * 60)
    
    if not rules:
        info("Không có rule nào")
        print()
        return
    
    # Lấy role names map: ưu tiên dùng tham số truyền vào, nếu không có hoặc rỗng thì gọi API
    if role_names_map is None or len(role_names_map) == 0:
        role_names_map = {}
        if token:
            role_names_map = get_role_names_map(token)
    
    for rule in rules:
        rule_id = rule.get("id", "N/A")
        rule_type = rule.get("type", "N/A")
        fixed = rule.get("fixed", False)
        service_name = rule.get("service_name") or ""
        roles = rule.get("roles", [])
        
        # Convert role IDs sang role names
        role_names = []
        for role_id in roles:
            if role_id in role_names_map:
                role_names.append(f'"{role_names_map[role_id]}"')
            else:
                # Nếu không tìm thấy name, dùng ID
                role_names.append(f'"{role_id}"')
        
        # Format roles string
        roles_str = ", ".join(role_names) if role_names else ""
        type_with_roles = f'{rule_type}({roles_str})' if roles_str else rule_type
        
        # Format theo yêu cầu: ID  , TYPE("role1", "role2") , fixed, service_name
        # Nếu fixed = false thì không hiển thị "fixed"
        # Nếu service_name rỗng thì không hiển thị
        output = f"{rule_id}  , {type_with_roles}"
        
        # Thêm fixed nếu có
        if fixed:
            output += " , fixed"
        
        # Thêm service_name nếu có
        if service_name:
            if fixed:
                # Nếu đã có fixed, dùng dấu phẩy không có space trước
                output += f", {service_name}"
            else:
                # Nếu chưa có fixed, dùng format giống sau type
                output += f" , {service_name}"
        
        print(output)
    
    print()

def login_safe(email: str, password: str) -> Tuple[bool, Optional[str], Optional[str]]:
    """
    Login mà không exit chương trình, trả về tuple để xử lý lỗi
    
    Args:
        email: Email để login
        password: Password để login
    
    Returns:
        Tuple (success, token, error_message)
        - success: True nếu login thành công, False nếu lỗi
        - token: JWT token nếu thành công, None nếu lỗi
        - error_message: Thông báo lỗi nếu có, None nếu thành công
    """
    try:
        token, _ = login(email, password)
        return True, token, None
    except SystemExit:
        return False, None, "Đăng nhập thất bại"
    except Exception as e:
        return False, None, f"Lỗi không mong đợi: {str(e)}"

def delete_user(token: str, user_id: str) -> Tuple[bool, Optional[str]]:
    """
    Hard delete user bằng token (thường là super_admin hoặc admin)
    
    Args:
        token: JWT token để xác thực
        user_id: ID của user cần xóa
    
    Returns:
        Tuple (success, error_message)
        - success: True nếu xóa thành công, False nếu lỗi
        - error_message: Thông báo lỗi nếu có, None nếu thành công
    """
    try:
        info(f"Đang xóa user ID: {user_id}...")
        resp = requests.delete(
            f"{_BASE_URL}/api/auth/profile/{user_id}",
            headers={"Authorization": f"Bearer {token}"},
            timeout=10
        )
        
        # Parse response
        try:
            resp_data = resp.json()
        except json.JSONDecodeError:
            return False, f"Response không phải JSON. Status: {resp.status_code}"
        
        # Kiểm tra lỗi
        if resp.status_code != 200:
            error_msg = "Lỗi xóa user không xác định"
            
            # Thử lấy từ "error" object (nếu là dict)
            error_obj = resp_data.get("error")
            if isinstance(error_obj, dict):
                error_msg = error_obj.get("message", error_msg)
            elif isinstance(error_obj, str):
                error_msg = error_obj
            
            # Thử lấy từ top level "message" (format của goerrorkit)
            if "message" in resp_data:
                error_msg = resp_data.get("message", error_msg)
            
            return False, error_msg
        
        success(f"Xóa user ID {user_id} thành công!")
        return True, None
        
    except requests.exceptions.RequestException as e:
        return False, f"Lỗi kết nối: {str(e)}"
    except Exception as e:
        return False, f"Lỗi không mong đợi: {str(e)}"

def confirm_reset(action_description: str = "reset dữ liệu", warning_message: str = None) -> bool:
    """
    Đợi người dùng xác nhận có muốn reset dữ liệu hay không
    Hàm dùng chung để xác nhận các thao tác reset/xóa dữ liệu
    
    Args:
        action_description: Mô tả hành động sẽ thực hiện (ví dụ: "xóa các user", "reset roles")
        warning_message: Thông báo cảnh báo tùy chỉnh. Nếu None, sẽ dùng thông báo mặc định
    
    Returns:
        True nếu người dùng xác nhận, False nếu không
    """
    print()
    print("=" * 80)
    info(f"XÁC NHẬN {action_description.upper()}")
    print("=" * 80)
    print()
    print(f"Bạn có muốn {action_description} không?")
    
    if warning_message:
        print(f"⚠️  {warning_message}")
    else:
        print("⚠️  Lưu ý: Đây là thao tác HARD DELETE, không thể hoàn tác!")
    print()
    
    while True:
        try:
            response = input("Nhập 'y' hoặc 'yes' để xác nhận, 'n' hoặc 'no' để hủy: ").strip().lower()
            
            if response in ['y', 'yes']:
                return True
            elif response in ['n', 'no']:
                return False
            else:
                error("Vui lòng nhập 'y'/'yes' hoặc 'n'/'no'")
                print()
        except KeyboardInterrupt:
            print()
            info("Đã hủy bởi người dùng.")
            return False
        except EOFError:
            print()
            info("Đã hủy bởi người dùng.")
            return False
