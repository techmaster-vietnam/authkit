# API Spec: List Users

## Endpoint
```
GET /api/user/
```

## Mô tả
Lấy danh sách users với hỗ trợ pagination, filtering, sorting và custom fields. Chỉ dành cho admin và super_admin.

## Authentication
- **Required**: Có
- **Type**: Bearer Token (JWT)
- **Header**: `Authorization: Bearer <token>`

## Authorization
- **Required Roles**: `admin` hoặc `super_admin`

## Query Parameters

### Pagination Parameters

| Parameter | Type | Default | Mô tả |
|-----------|------|---------|-------|
| `page` | integer | `1` | Số trang (bắt đầu từ 1) |
| `page_size` | integer | `40` | Số lượng items mỗi trang |
| `enable_pagination` | string | `auto` | Bật/tắt pagination: `"true"`, `"false"`, hoặc `auto` (tự động dựa trên threshold) |
| `pagination_threshold` | integer | `100` | Ngưỡng để tự động bật pagination (nếu total > threshold thì bật pagination) |
| `max_page_size` | integer | `100` | Giới hạn tối đa page_size khi có pagination |

### Sorting Parameters

| Parameter | Type | Default | Mô tả |
|-----------|------|---------|-------|
| `sort_by` | string | - | Trường để sort (ví dụ: `email`, `full_name`, `mobile`, `address`, hoặc custom field) |
| `order` | string | `asc` | Thứ tự sort: `"asc"` hoặc `"desc"` |

### Filter Parameters

| Parameter | Type | Mô tả |
|-----------|------|-------|
| `email` | string | Filter email chứa text (partial match) |
| `full_name` | string | Filter full_name chứa text (partial match) |
| `role_name` | string | Filter users có role chứa text (partial match). Chỉ trả về users có ít nhất một role có tên chứa text này |
| `<custom_field>` | string | Filter theo custom field (ví dụ: `mobile`, `address`, v.v.) |

**Lưu ý**: 
- Tất cả các query parameter khác ngoài các parameter đã liệt kê ở trên sẽ được coi là custom field filters.
- Filter `role_name` sẽ tìm kiếm trong danh sách roles của user. Nếu user có nhiều roles, chỉ cần một role khớp là user sẽ được trả về.

## Response Format

### Success Response (200 OK)

```json
{
  "data": {
    "users": [
      {
        "id": "user_id_1",
        "email": "user1@example.com",
        "full_name": "Nguyễn Văn A",
        "roles": ["admin", "editor"],
        "mobile": "0901234567",
        "address": "123 Đường ABC, Quận XYZ"
      },
      {
        "id": "user_id_2",
        "email": "user2@example.com",
        "full_name": "Trần Thị B",
        "roles": ["reader"],
        "mobile": "0987654321"
      }
    ],
    "total": 150,
    "pagination_enabled": true,
    "page": 1,
    "page_size": 40,
    "total_pages": 4
  }
}
```

### Response khi không có pagination

```json
{
  "data": {
    "users": [
      {
        "id": "user_id_1",
        "email": "user1@example.com",
        "full_name": "Nguyễn Văn A",
        "roles": []
      }
    ],
    "total": 25,
    "pagination_enabled": false
  }
}
```

### Error Response (401 Unauthorized)

```json
{
  "error": "Unauthorized",
  "message": "Token không hợp lệ hoặc đã hết hạn"
}
```

### Error Response (403 Forbidden)

```json
{
  "error": "Forbidden",
  "message": "Bạn không có quyền truy cập endpoint này"
}
```

## Response Fields

### Data Object

| Field | Type | Mô tả |
|-------|------|-------|
| `users` | array | Danh sách user objects |
| `total` | integer | Tổng số users (không phân trang) |
| `pagination_enabled` | boolean | `true` nếu đang dùng pagination, `false` nếu không |
| `page` | integer (optional) | Số trang hiện tại (chỉ có khi `pagination_enabled = true`) |
| `page_size` | integer (optional) | Số lượng items mỗi trang (chỉ có khi `pagination_enabled = true`) |
| `total_pages` | integer (optional) | Tổng số trang (chỉ có khi `pagination_enabled = true`) |

### User Object

| Field | Type | Mô tả |
|-------|------|-------|
| `id` | string | User ID (unique identifier) |
| `email` | string | Email của user |
| `full_name` | string | Họ và tên đầy đủ |
| `roles` | array[string] | Danh sách tên roles của user (có thể rỗng `[]`) |
| `<custom_field>` | string/number/boolean | Các custom fields khác (ví dụ: `mobile`, `address`, v.v.) |

## Ví dụ sử dụng với Vue.js

### 1. Component Vue.js cơ bản

```vue
<template>
  <div>
    <h2>Danh sách Users</h2>
    
    <!-- Filter form -->
    <div class="filters">
      <input v-model="filters.email" placeholder="Email" @input="loadUsers" />
      <input v-model="filters.full_name" placeholder="Họ tên" @input="loadUsers" />
      <input v-model="filters.role_name" placeholder="Role name" @input="loadUsers" />
      <select v-model="filters.sort_by" @change="loadUsers">
        <option value="">Không sort</option>
        <option value="email">Email</option>
        <option value="full_name">Họ tên</option>
      </select>
      <select v-model="filters.order" @change="loadUsers">
        <option value="asc">Tăng dần</option>
        <option value="desc">Giảm dần</option>
      </select>
    </div>

    <!-- User list -->
    <div v-if="loading">Đang tải...</div>
    <div v-else>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Email</th>
            <th>Họ tên</th>
            <th>Roles</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.id }}</td>
            <td>{{ user.email }}</td>
            <td>{{ user.full_name }}</td>
            <td>{{ user.roles.join(', ') }}</td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="paginationEnabled" class="pagination">
        <button @click="prevPage" :disabled="page === 1">Trước</button>
        <span>Trang {{ page }}/{{ totalPages }} (Tổng: {{ total }} users)</span>
        <button @click="nextPage" :disabled="page >= totalPages">Sau</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const users = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(40)
const totalPages = ref(0)
const paginationEnabled = ref(false)

const filters = ref({
  email: '',
  full_name: '',
  role_name: '',
  sort_by: '',
  order: 'asc'
})

const loadUsers = async () => {
  loading.value = true
  try {
    const token = localStorage.getItem('token') // Hoặc lấy từ store/auth
    
    const params = {
      page: page.value,
      page_size: pageSize.value,
      ...filters.value
    }
    
    // Loại bỏ các filter rỗng
    Object.keys(params).forEach(key => {
      if (params[key] === '') delete params[key]
    })

    const response = await axios.get('/api/user/', {
      params,
      headers: {
        Authorization: `Bearer ${token}`
      }
    })

    const data = response.data.data
    users.value = data.users
    total.value = data.total
    paginationEnabled.value = data.pagination_enabled
    
    if (data.pagination_enabled) {
      page.value = data.page
      pageSize.value = data.page_size
      totalPages.value = data.total_pages
    }
  } catch (error) {
    console.error('Lỗi khi tải danh sách users:', error)
    if (error.response?.status === 401) {
      // Redirect to login
    } else if (error.response?.status === 403) {
      // Show forbidden message
    }
  } finally {
    loading.value = false
  }
}

const nextPage = () => {
  if (page.value < totalPages.value) {
    page.value++
    loadUsers()
  }
}

const prevPage = () => {
  if (page.value > 1) {
    page.value--
    loadUsers()
  }
}

onMounted(() => {
  loadUsers()
})
</script>
```

### 2. Composable function (Vue 3 Composition API)

```javascript
// composables/useUserList.js
import { ref } from 'vue'
import axios from 'axios'

export function useUserList() {
  const users = ref([])
  const loading = ref(false)
  const error = ref(null)
  const pagination = ref({
    total: 0,
    page: 1,
    pageSize: 40,
    totalPages: 0,
    enabled: false
  })

  const loadUsers = async (options = {}) => {
    loading.value = true
    error.value = null
    
    try {
      const token = localStorage.getItem('token')
      
      const params = {
        page: options.page || pagination.value.page,
        page_size: options.pageSize || pagination.value.pageSize,
        email: options.email || '',
        full_name: options.fullName || '',
        role_name: options.roleName || '',
        sort_by: options.sortBy || '',
        order: options.order || 'asc',
        ...options.customFilters
      }
      
      // Remove empty params
      Object.keys(params).forEach(key => {
        if (params[key] === '') delete params[key]
      })

      const response = await axios.get('/api/user/', {
        params,
        headers: {
          Authorization: `Bearer ${token}`
        }
      })

      const data = response.data.data
      users.value = data.users
      pagination.value.total = data.total
      pagination.value.enabled = data.pagination_enabled
      
      if (data.pagination_enabled) {
        pagination.value.page = data.page
        pagination.value.pageSize = data.page_size
        pagination.value.totalPages = data.total_pages
      }
      
      return data
    } catch (err) {
      error.value = err.response?.data || err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    users,
    loading,
    error,
    pagination,
    loadUsers
  }
}
```

### 3. Sử dụng composable

```vue
<template>
  <div>
    <UserListFilters @filter="handleFilter" />
    <UserListTable :users="users" :loading="loading" />
    <UserListPagination 
      v-if="pagination.enabled"
      :pagination="pagination"
      @page-change="handlePageChange"
    />
  </div>
</template>

<script setup>
import { useUserList } from '@/composables/useUserList'

const { users, loading, error, pagination, loadUsers } = useUserList()

const handleFilter = (filters) => {
  pagination.value.page = 1 // Reset về trang đầu
  loadUsers(filters)
}

const handlePageChange = (newPage) => {
  loadUsers({ page: newPage })
}

// Load initial data
loadUsers()
</script>
```

### 4. Ví dụ sử dụng filter by role name

```javascript
// Lấy danh sách users có role "admin"
const response = await axios.get('/api/user/', {
  params: {
    role_name: 'admin',
    page: 1,
    page_size: 40
  },
  headers: {
    Authorization: `Bearer ${token}`
  }
})

// Lấy danh sách users có role chứa "tiger", sort theo email A-Z
const response2 = await axios.get('/api/user/', {
  params: {
    role_name: 'tiger',
    sort_by: 'email',
    order: 'asc',
    page: 1,
    page_size: 10
  },
  headers: {
    Authorization: `Bearer ${token}`
  }
})

// Kết hợp nhiều filters: email chứa "micro" và có role "bird"
const response3 = await axios.get('/api/user/', {
  params: {
    email: 'micro',
    role_name: 'bird',
    sort_by: 'email',
    order: 'desc'
  },
  headers: {
    Authorization: `Bearer ${token}`
  }
})
```

## Lưu ý

1. **Pagination tự động**: Nếu `enable_pagination` không được chỉ định hoặc là `auto`, hệ thống sẽ tự động quyết định có dùng pagination hay không dựa trên `pagination_threshold`.

2. **Custom fields**: Response sẽ tự động bao gồm tất cả các custom fields của User model (ví dụ: `mobile`, `address`). Các field này có thể được dùng để filter và sort.

3. **Roles**: Mỗi user object luôn có field `roles` là một array (có thể rỗng `[]`). Có thể filter users theo role name bằng parameter `role_name`. Filter này sẽ tìm kiếm partial match trong danh sách roles của user.

4. **Error handling**: Luôn kiểm tra status code và xử lý các trường hợp:
   - `401`: Token không hợp lệ hoặc đã hết hạn → Redirect to login
   - `403`: Không có quyền truy cập → Hiển thị thông báo lỗi
   - `500`: Lỗi server → Hiển thị thông báo lỗi

5. **Performance**: Khi có nhiều users, nên sử dụng pagination để tối ưu performance. Mặc định `page_size = 40` là hợp lý cho hầu hết các trường hợp.

