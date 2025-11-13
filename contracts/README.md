# Contracts Package

Package `contracts` định nghĩa các interfaces cho hệ thống authentication và authorization của AuthKit.

## Mục đích

AuthKit sử dụng **Interface Segregation Principle** và **Dependency Inversion Principle** từ SOLID để cho phép ứng dụng bên ngoài implement các interfaces này với models và repositories của riêng họ.

## Các Interfaces

### UserInterface

Định nghĩa cấu trúc tối thiểu cần thiết cho User trong hệ thống authentication:

```go
type UserInterface interface {
    GetID() uuid.UUID
    GetEmail() string
    GetUsername() string
    GetPassword() string
    SetPassword(password string)
    IsActive() bool
    SetActive(active bool)
    GetRoles() []RoleInterface
}
```

### RoleInterface

Định nghĩa cấu trúc tối thiểu cần thiết cho Role:

```go
type RoleInterface interface {
    GetID() uuid.UUID
    GetName() string
    IsSystem() bool
}
```

### RuleInterface

Định nghĩa cấu trúc tối thiểu cần thiết cho Rule (authorization rules):

```go
type RuleInterface interface {
    GetID() uuid.UUID
    GetMethod() string
    GetPath() string
    GetType() RuleType
    GetRoles() []string
    GetPriority() int
}
```

### Repository Interfaces

- `UserRepositoryInterface`: Interface cho User Repository
- `RoleRepositoryInterface`: Interface cho Role Repository  
- `RuleRepositoryInterface`: Interface cho Rule Repository

## Cách sử dụng

### Option 1: Sử dụng Reference Implementation

Package `models` cung cấp reference implementation của các interfaces. Bạn có thể sử dụng trực tiếp:

```go
import "github.com/techmaster-vietnam/authkit/models"

user := &models.User{
    Email: "user@example.com",
    // ...
}
```

### Option 2: Implement Interfaces với Model của riêng bạn

Nếu bạn có model User riêng với các trường bổ sung, bạn có thể implement `UserInterface`:

```go
type MyUser struct {
    ID       uuid.UUID
    Email    string
    Username string
    Password string
    Active   bool
    // Các trường khác của bạn
    CompanyID uuid.UUID
    // ...
}

func (u *MyUser) GetID() uuid.UUID { return u.ID }
func (u *MyUser) GetEmail() string { return u.Email }
// ... implement các methods khác
```

Sau đó implement `UserRepositoryInterface` với repository của bạn để làm việc với `MyUser`.

## Lưu ý

- Models trong package `models` chỉ là **reference implementation** để tham khảo
- Ứng dụng có thể extend models này hoặc implement interfaces với models hoàn toàn mới
- AuthKit không quản lý database migration - ứng dụng tự chịu trách nhiệm migrate database
- AuthKit chỉ định nghĩa contracts/interfaces, không ép buộc cấu trúc database cụ thể

