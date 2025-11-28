# Đề xuất: File Log Level cho goerrorkit

## 1. Vấn đề hiện tại

Hiện tại, goerrorkit log tất cả errors vào file `errors.log` bất kể mức độ nghiêm trọng. Điều này dẫn đến:

- **File log bị nhiễu** với các validation errors không nghiêm trọng (ví dụ: email không hợp lệ, password quá ngắn)
- **Khó phân tích** các lỗi thực sự nghiêm trọng trong production
- **Tốn dung lượng** lưu trữ với các log không cần thiết
- **Không tuân theo best practices** của logging: chỉ log các lỗi nghiêm trọng vào file

### Ví dụ vấn đề:

Khi user đăng ký với email/password không hợp lệ, các validation errors được log vào file:
```json
{
  "error_type": "VALIDATION",
  "level": "error",
  "message": "Email không hợp lệ: thiếu ký tự @",
  "path": "POST /api/auth/register",
  "timestamp": "2025-11-28T14:42:38+07:00"
}
```

Đây là lỗi **bình thường** trong flow của ứng dụng, không cần ghi vào file log.

## 2. Giải pháp đề xuất

Thêm khả năng **phân cấp log level** để quyết định:
- **Console**: Log tất cả errors (theo `LogLevel`)
- **File**: Chỉ log các errors có level >= `FileLogLevel`

### Log Level Hierarchy:
```
info < warning < error < fatal
```

### Mặc định log level cho các error types:
- `ValidationError` → `"warning"` (không nghiêm trọng)
- `BusinessError` → `"error"` (nghiêm trọng)
- `AuthError` → `"error"` (nghiêm trọng)
- `SystemError` → `"error"` (nghiêm trọng)

## 3. Thay đổi API

### 3.1. Thêm field `FileLogLevel` vào `LoggerOptions`

```go
type LoggerOptions struct {
    ConsoleOutput bool
    FileOutput    bool
    FilePath      string
    JSONFormat    bool
    MaxFileSize   int
    MaxBackups    int
    MaxAge        int
    LogLevel      string  // Level tối thiểu để log (console + file)
    
    // NEW: Level tối thiểu để ghi vào file
    // Chỉ các error có level >= FileLogLevel mới được ghi vào file
    // Các error có level < FileLogLevel chỉ log console
    // Mặc định: "error" (chỉ error và fatal ghi vào file)
    // Giá trị hợp lệ: "info", "warning", "error", "fatal"
    FileLogLevel  string
}
```

### 3.2. Thêm method `WithLevel()` cho các error types

```go
// ValidationError
func (e *ValidationError) WithLevel(level string) *ValidationError

// BusinessError  
func (e *BusinessError) WithLevel(level string) *BusinessError

// AuthError
func (e *AuthError) WithLevel(level string) *AuthError

// SystemError
func (e *SystemError) WithLevel(level string) *SystemError
```

### 3.3. Thêm field `Level` vào struct `AppError`

```go
type AppError struct {
    // ... existing fields ...
    Level string  // "info", "warning", "error", "fatal"
}
```

## 4. Triển khai chi tiết

### 4.1. Constants cho log levels

```go
const (
    LogLevelInfo    = "info"
    LogLevelWarning = "warning"
    LogLevelError   = "error"
    LogLevelFatal   = "fatal"
)

var logLevelOrder = map[string]int{
    LogLevelInfo:    0,
    LogLevelWarning: 1,
    LogLevelError:   2,
    LogLevelFatal:   3,
}
```

### 4.2. Helper function để so sánh log levels

```go
// compareLogLevels trả về:
// -1 nếu level1 < level2
// 0 nếu level1 == level2
// 1 nếu level1 > level2
func compareLogLevels(level1, level2 string) int {
    order1, ok1 := logLevelOrder[strings.ToLower(level1)]
    order2, ok2 := logLevelOrder[strings.ToLower(level2)]
    
    if !ok1 {
        order1 = logLevelOrder[LogLevelError] // Default to error
    }
    if !ok2 {
        order2 = logLevelOrder[LogLevelError] // Default to error
    }
    
    if order1 < order2 {
        return -1
    } else if order1 > order2 {
        return 1
    }
    return 0
}

// shouldLogToFile kiểm tra xem error có nên log vào file không
func shouldLogToFile(errorLevel, fileLogLevel string) bool {
    return compareLogLevels(errorLevel, fileLogLevel) >= 0
}
```

### 4.3. Set default level cho các error types

```go
// Trong NewValidationError
func NewValidationError(message string, data map[string]interface{}) *ValidationError {
    return &ValidationError{
        AppError: AppError{
            Type:    ErrorTypeValidation,
            Message: message,
            Data:    data,
            Level:   LogLevelWarning, // Default: warning
        },
    }
}

// Trong NewBusinessError
func NewBusinessError(code int, message string) *BusinessError {
    return &BusinessError{
        AppError: AppError{
            Type:    ErrorTypeBusiness,
            Code:    code,
            Message: message,
            Level:   LogLevelError, // Default: error
        },
    }
}

// Trong NewAuthError
func NewAuthError(code int, message string) *AuthError {
    return &AuthError{
        AppError: AppError{
            Type:    ErrorTypeAuth,
            Code:    code,
            Message: message,
            Level:   LogLevelError, // Default: error
        },
    }
}

// Trong NewSystemError
func NewSystemError(err error) *SystemError {
    return &SystemError{
        AppError: AppError{
            Type:    ErrorTypeSystem,
            Message: err.Error(),
            Cause:   err,
            Level:   LogLevelError, // Default: error
        },
    }
}
```

### 4.4. Implement `WithLevel()` method

```go
// Cho ValidationError
func (e *ValidationError) WithLevel(level string) *ValidationError {
    e.Level = strings.ToLower(level)
    return e
}

// Cho BusinessError
func (e *BusinessError) WithLevel(level string) *BusinessError {
    e.Level = strings.ToLower(level)
    return e
}

// Cho AuthError
func (e *AuthError) WithLevel(level string) *AuthError {
    e.Level = strings.ToLower(level)
    return e
}

// Cho SystemError
func (e *SystemError) WithLevel(level string) *SystemError {
    e.Level = strings.ToLower(level)
    return e
}
```

### 4.5. Update `InitLogger()` để set default `FileLogLevel`

```go
func InitLogger(options LoggerOptions) {
    // Set default FileLogLevel nếu không được chỉ định
    if options.FileLogLevel == "" {
        options.FileLogLevel = LogLevelError // Default: chỉ log error và fatal vào file
    }
    
    // Validate FileLogLevel
    if _, ok := logLevelOrder[strings.ToLower(options.FileLogLevel)]; !ok {
        options.FileLogLevel = LogLevelError // Fallback to error
    }
    
    // ... existing initialization code ...
    
    // Store FileLogLevel globally để sử dụng khi log
    globalFileLogLevel = strings.ToLower(options.FileLogLevel)
}
```

### 4.6. Update `LogError()` để check FileLogLevel

```go
func LogError(err *AppError, context string) {
    // Luôn log ra console (theo LogLevel)
    if shouldLogToConsole(err.Level, globalLogLevel) {
        logToConsole(err, context)
    }
    
    // Chỉ log vào file nếu error level >= FileLogLevel
    if shouldLogToFile(err.Level, globalFileLogLevel) {
        logToFile(err, context)
    }
}
```

### 4.7. Update `FiberErrorHandler()` để sử dụng level

```go
func FiberErrorHandler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        err := c.Next()
        if err != nil {
            appErr := convertToAppError(err)
            
            // Log error với level
            LogError(appErr, fmt.Sprintf("%s %s", c.Method(), c.Path()))
            
            // Return response
            return c.Status(getStatusCode(appErr)).
                JSON(formatErrorResponse(appErr))
        }
        return nil
    }
}
```

## 5. Cách sử dụng

### 5.1. Cấu hình cơ bản

```go
goerrorkit.InitLogger(goerrorkit.LoggerOptions{
    ConsoleOutput: true,
    FileOutput:    true,
    FilePath:      "logs/errors.log",
    JSONFormat:    true,
    MaxFileSize:   10,
    MaxBackups:    5,
    MaxAge:        30,
    LogLevel:      "info",      // Log tất cả từ info trở lên (console + file)
    FileLogLevel:  "error",     // Chỉ ghi vào file từ error trở lên
})
```

**Kết quả:**
- `ValidationError` (warning) → Chỉ log console, không log file
- `SystemError` (error) → Log cả console và file

### 5.2. Override level khi tạo error

```go
// ValidationError mặc định là "warning", có thể override
err := goerrorkit.NewValidationError("Email không hợp lệ").
    WithLevel("info").  // Override thành "info"
    WithData(map[string]interface{}{
        "field": "email",
    })

// SystemError mặc định là "error", có thể override
err := goerrorkit.NewSystemError(originalErr).
    WithLevel("fatal").  // Override thành "fatal"
    WithData(...)
```

### 5.3. Các trường hợp sử dụng

#### Case 1: Chỉ log error và fatal vào file
```go
FileLogLevel: "error"
```
- `info`, `warning` → Chỉ console
- `error`, `fatal` → Console + file

#### Case 2: Log tất cả vào file
```go
FileLogLevel: "info"
```
- Tất cả levels → Console + file

#### Case 3: Chỉ log fatal vào file
```go
FileLogLevel: "fatal"
```
- `info`, `warning`, `error` → Chỉ console
- `fatal` → Console + file

## 6. Migration Guide

### 6.1. Backward Compatibility

- **FileLogLevel mặc định là "error"** → Không breaking change
- Các error types mặc định có level phù hợp → Không cần thay đổi code hiện tại
- Nếu không set `FileLogLevel`, behavior giống như trước (log tất cả vào file nếu `LogLevel` cho phép)

### 6.2. Để giữ behavior cũ (log tất cả vào file)

```go
goerrorkit.InitLogger(goerrorkit.LoggerOptions{
    // ... other options ...
    FileLogLevel: "info",  // Log tất cả vào file
})
```

### 6.3. Để filter validation errors khỏi file

```go
goerrorkit.InitLogger(goerrorkit.LoggerOptions{
    // ... other options ...
    FileLogLevel: "error",  // Chỉ log error và fatal vào file
})
```

## 7. Test Cases

### 7.1. Test log level comparison

```go
func TestCompareLogLevels(t *testing.T) {
    tests := []struct {
        level1   string
        level2   string
        expected int
    }{
        {"info", "warning", -1},
        {"warning", "error", -1},
        {"error", "error", 0},
        {"error", "warning", 1},
        {"fatal", "error", 1},
    }
    
    for _, tt := range tests {
        result := compareLogLevels(tt.level1, tt.level2)
        assert.Equal(t, tt.expected, result)
    }
}
```

### 7.2. Test shouldLogToFile

```go
func TestShouldLogToFile(t *testing.T) {
    tests := []struct {
        errorLevel   string
        fileLogLevel string
        expected     bool
    }{
        {"warning", "error", false},  // warning < error
        {"error", "error", true},      // error == error
        {"fatal", "error", true},      // fatal > error
        {"info", "warning", false},    // info < warning
    }
    
    for _, tt := range tests {
        result := shouldLogToFile(tt.errorLevel, tt.fileLogLevel)
        assert.Equal(t, tt.expected, result)
    }
}
```

### 7.3. Test default levels

```go
func TestDefaultErrorLevels(t *testing.T) {
    valErr := goerrorkit.NewValidationError("test")
    assert.Equal(t, "warning", valErr.Level)
    
    sysErr := goerrorkit.NewSystemError(errors.New("test"))
    assert.Equal(t, "error", sysErr.Level)
    
    bizErr := goerrorkit.NewBusinessError(400, "test")
    assert.Equal(t, "error", bizErr.Level)
    
    authErr := goerrorkit.NewAuthError(401, "test")
    assert.Equal(t, "error", authErr.Level)
}
```

### 7.4. Test WithLevel override

```go
func TestWithLevelOverride(t *testing.T) {
    valErr := goerrorkit.NewValidationError("test").
        WithLevel("error")
    assert.Equal(t, "error", valErr.Level)
    
    sysErr := goerrorkit.NewSystemError(errors.New("test")).
        WithLevel("fatal")
    assert.Equal(t, "fatal", sysErr.Level)
}
```

### 7.5. Integration test: ValidationError không log vào file

```go
func TestValidationErrorNotLoggedToFile(t *testing.T) {
    // Setup logger với FileLogLevel = "error"
    goerrorkit.InitLogger(goerrorkit.LoggerOptions{
        ConsoleOutput: true,
        FileOutput:    true,
        FilePath:      "test_errors.log",
        FileLogLevel:  "error",
    })
    
    // Create validation error (default level = "warning")
    err := goerrorkit.NewValidationError("Email không hợp lệ")
    goerrorkit.LogError(err, "test")
    
    // Kiểm tra file không chứa validation error
    content, _ := os.ReadFile("test_errors.log")
    assert.NotContains(t, string(content), "Email không hợp lệ")
    
    // Cleanup
    os.Remove("test_errors.log")
}
```

## 8. Ví dụ thực tế

### 8.1. Trước khi có FileLogLevel

```go
// Tất cả errors đều log vào file
goerrorkit.InitLogger(goerrorkit.LoggerOptions{
    FileOutput: true,
    FilePath:   "logs/errors.log",
    LogLevel:   "info",
})

// ValidationError → Log vào file ❌ (không mong muốn)
err := goerrorkit.NewValidationError("Email không hợp lệ")
goerrorkit.LogError(err, "register")
```

### 8.2. Sau khi có FileLogLevel

```go
// Chỉ log error và fatal vào file
goerrorkit.InitLogger(goerrorkit.LoggerOptions{
    FileOutput:   true,
    FilePath:     "logs/errors.log",
    LogLevel:     "info",
    FileLogLevel: "error",  // NEW
})

// ValidationError (warning) → Chỉ log console ✅
err := goerrorkit.NewValidationError("Email không hợp lệ")
goerrorkit.LogError(err, "register")

// SystemError (error) → Log cả console và file ✅
sysErr := goerrorkit.NewSystemError(databaseErr)
goerrorkit.LogError(sysErr, "database")
```

## 9. Checklist triển khai

- [ ] Thêm constants cho log levels
- [ ] Thêm field `Level` vào `AppError`
- [ ] Thêm field `FileLogLevel` vào `LoggerOptions`
- [ ] Implement `compareLogLevels()` và `shouldLogToFile()`
- [ ] Set default level cho các error types
- [ ] Implement `WithLevel()` cho tất cả error types
- [ ] Update `InitLogger()` để set default `FileLogLevel`
- [ ] Update `LogError()` để check `FileLogLevel`
- [ ] Update `FiberErrorHandler()` nếu cần
- [ ] Viết unit tests
- [ ] Viết integration tests
- [ ] Update documentation
- [ ] Update examples

## 10. Lưu ý

1. **Case-insensitive**: Log levels nên được normalize thành lowercase
2. **Invalid level**: Nếu level không hợp lệ, default về "error"
3. **Empty FileLogLevel**: Mặc định là "error"
4. **Performance**: So sánh log levels nên nhanh (dùng map lookup)
5. **JSON output**: Level nên được include trong JSON log output

## 11. Breaking Changes

**Không có breaking changes** nếu:
- `FileLogLevel` mặc định là "error"
- Các error types có default level phù hợp
- Behavior cũ vẫn hoạt động nếu set `FileLogLevel: "info"`

**Có breaking change** nếu:
- User muốn log validation errors vào file → Cần set `FileLogLevel: "info"` hoặc override level của ValidationError thành "error"

