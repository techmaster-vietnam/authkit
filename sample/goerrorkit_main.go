package main

import (
	"fmt"

	fiberv2 "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/techmaster-vietnam/goerrorkit"
	"github.com/techmaster-vietnam/goerrorkit/adapters/fiber"
)

func main() {
	// 1. Initialize logger với custom options
	goerrorkit.InitLogger(goerrorkit.LoggerOptions{
		ConsoleOutput: true,
		FileOutput:    true,
		FilePath:      "logs/errors.log",
		JSONFormat:    true,
		MaxFileSize:   10,
		MaxBackups:    5,
		MaxAge:        30,
		LogLevel:      "info",
	})

	// 2. Configure stack trace for this application
	// 🎯 MỤC ĐÍCH: Lọc stack trace để CHỈ HIỂN THỊ code của BẠN, bỏ qua:
	//    - Go runtime code (runtime.*, runtime/debug.*)
	//    - Thư viện bên thứ 3 (fiber, goerrorkit, etc.)
	//
	// ✅ CÁCH DÙNG:
	//    - App đơn giản (1 file main.go):
	//      goerrorkit.ConfigureForApplication("main")
	//
	//    - App với nhiều package (services/, handlers/, models/...):
	//      goerrorkit.ConfigureForApplication("github.com/techmaster-vietnam/goerrorkit/examples/fiber-demo")
	//      → Tự động include TẤT CẢ sub-packages!
	//
	// 📊 KẾT QUẢ:
	//    KHÔNG cấu hình: Stack trace dài 50+ dòng (runtime, fiber, goerrorkit...)
	//    CÓ cấu hình:    Stack trace ngắn gọn, chỉ 5-10 dòng CODE CỦA BẠN!
	//
	goerrorkit.ConfigureForApplication("main")

	// 🔧 FLUENT API: Nếu cần thêm các patterns tùy chỉnh, có thể dùng:
	//
	// Cách 1: Shorthand - Nhanh chóng thêm skip patterns
	// goerrorkit.AddSkipPatterns(".RequestID.func", ".Logger.func", "telemetry")
	//
	// Cách 2: Fluent API - Configuration chi tiết hơn
	// goerrorkit.Configure().
	//     SkipPattern(".CustomMiddleware.func").
	//     SkipPackage("internal/metrics").
	//     SkipFunctions("helper", "wrapper").
	//     ShowFullPath(false).
	//     Apply()

	// 3. Create Fiber app
	app := fiberv2.New(fiberv2.Config{
		AppName: "GoErrorKit Demo",
	})

	// 4. Add middlewares (RequestID must be before ErrorHandler)
	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(fiber.ErrorHandler())

	// 5. Routes - Demo different error types
	app.Get("/", homeHandler)
	app.Get("/favicon.ico", faviconHandler) // Serve favicon
	app.Get("/favicon.svg", faviconHandler) // Modern SVG favicon

	// Panic demos
	app.Get("/panic/division", panicDivisionHandler)
	app.Get("/panic/index", panicIndexHandler)
	app.Get("/panic/stack", panicStackHandler)

	// Wrap error demos (NEW!)
	app.Get("/error/wrap", wrapErrorHandler)
	app.Get("/error/wrap-message", wrapWithMessageHandler)

	// Custom error demos
	app.Get("/error/business", businessErrorHandler)
	app.Get("/error/system", systemErrorHandler)
	app.Get("/error/validation", validationErrorHandler)
	app.Get("/error/auth", authErrorHandler)
	app.Get("/error/external", externalErrorHandler)
	app.Get("/error/complex", complexErrorWithCallChainHandler)

	// Start server
	fmt.Println("🚀 Server starting on http://localhost:8081")
	fmt.Println("\n📝 Try these endpoints:")
	fmt.Println("  GET  /                     - Home")
	fmt.Println("\n  🔥 Panic Demos (auto-recovered with exact location):")
	fmt.Println("  GET  /panic/division       - Division by zero panic")
	fmt.Println("  GET  /panic/index          - Index out of range panic")
	fmt.Println("  GET  /panic/stack          - Deep call stack panic")
	fmt.Println("\n  🎁 Wrap Error Demos (NEW!):")
	fmt.Println("  GET  /error/wrap           - Wrap(err) - Đơn giản nhất")
	fmt.Println("  GET  /error/wrap-message   - WrapWithMessage(err, msg) - Thêm context")
	fmt.Println("\n  ⚠️  Custom Error Demos:")
	fmt.Println("  GET  /error/business       - Business logic error (404)")
	fmt.Println("  GET  /error/system         - System error (500)")
	fmt.Println("  GET  /error/validation     - Validation error (400)")
	fmt.Println("  GET  /error/auth           - Auth error (401)")
	fmt.Println("  GET  /error/external       - External service error (502)")
	fmt.Println("  GET  /error/complex        - Complex error WITH call_chain ⭐")
	fmt.Println("\n📄 Check logs/errors.log for detailed error logs")

	if err := app.Listen(":8081"); err != nil {
		panic(err)
	}
}

func homeHandler(c *fiberv2.Ctx) error {
	// Serve the index.html file
	return c.SendFile("./index.html")
}

func faviconHandler(c *fiberv2.Ctx) error {
	// Serve favicon.svg (modern browsers support SVG favicons)
	return c.SendFile("./favicon.svg")
}

// ============================================================================
// Panic Handlers - Demonstrate automatic panic recovery
// ============================================================================

func panicDivisionHandler(c *fiberv2.Ctx) error {
	// This will panic with "integer divide by zero"
	denominator := 0
	result := 100 / denominator // ← Panic location will be captured HERE!
	return c.JSON(fiberv2.Map{"result": result})
}

func panicIndexHandler(c *fiberv2.Ctx) error {
	// This will panic with "index out of range"
	element := GetElement() // Panic happens inside GetElement()
	return c.JSON(fiberv2.Map{"element": element})
}

func GetElement() int {
	arr := []int{1, 2, 3}
	return arr[10] // ← Panic location will be captured HERE!
}

func panicStackHandler(c *fiberv2.Ctx) error {
	// Deep call stack demo
	result := callX()
	return c.JSON(fiberv2.Map{"result": result})
}

func callX() int {
	return callY()
}

func callY() int {
	return callZ()
}

func callZ() int {
	return callW()
}

func callW() int {
	return GetElement() // Panic happens here, full call chain will be logged
}

// ============================================================================
// Wrap Error Handlers - Demonstrate Wrap() and WrapWithMessage()
// ============================================================================

// wrapErrorHandler demonstrates goerrorkit.Wrap(err)
// ✅ Use case: Wrap nhanh Go error với message gốc, tự động capture stack trace
func wrapErrorHandler(c *fiberv2.Ctx) error {
	errorType := c.Query("type", "json")

	switch errorType {
	case "json":
		// Simulate JSON parsing error
		err := fmt.Errorf("json: invalid character '}' looking for beginning of value")
		// ⭐ Wrap() - Đơn giản nhất, giữ nguyên message gốc
		// → Message: error message gốc
		// → Type: SystemError, Code: 500
		// → Tự động capture: file, line, function
		return goerrorkit.Wrap(err)

	case "database":
		// Simulate database connection error
		err := fmt.Errorf("sql: connection refused")
		// ⭐ Wrap() với error database
		return goerrorkit.Wrap(err)

	case "file":
		// Simulate file not found error
		err := fmt.Errorf("open config.json: no such file or directory")
		// ⭐ Wrap() với .WithData() - Thêm metadata
		return goerrorkit.Wrap(err).WithData(map[string]interface{}{
			"path":      "config.json",
			"operation": "read",
		})

	case "network":
		// Simulate network timeout
		err := fmt.Errorf("net/http: request timeout after 30s")
		// ⭐ Wrap() + WithData() + WithCallChain()
		return goerrorkit.Wrap(err).
			WithData(map[string]interface{}{
				"url":     "https://api.example.com/users",
				"timeout": "30s",
				"retries": 3,
			}).
			WithCallChain()
	}

	return c.JSON(fiberv2.Map{"message": "No error"})
}

// wrapWithMessageHandler demonstrates goerrorkit.WrapWithMessage(err, msg)
// ✅ Use case: Wrap error với custom message để thêm context, giữ error gốc trong Cause
func wrapWithMessageHandler(c *fiberv2.Ctx) error {
	scenario := c.Query("scenario", "database")

	switch scenario {
	case "database":
		// Simulate database query error
		err := fmt.Errorf("connection refused")
		// ⭐ WrapWithMessage() - Thêm context message
		// → Message: "Failed to fetch user list from database"
		// → Cause: "connection refused"
		// → Type: SystemError, Code: 500
		return goerrorkit.WrapWithMessage(err, "Failed to fetch user list from database")

	case "redis":
		// Simulate Redis cache error
		err := fmt.Errorf("redis: connection timeout")
		// ⭐ WrapWithMessage() với .WithData()
		return goerrorkit.WrapWithMessage(err, "Failed to get user session from cache").WithData(map[string]interface{}{
			"key": "user:session:12345",
			"ttl": 3600,
		})

	case "payment":
		// Simulate payment API error
		err := fmt.Errorf("stripe: card declined")
		// ⭐ WrapWithMessage() + WithData() - Detailed context
		return goerrorkit.WrapWithMessage(err, "Payment processing failed").WithData(map[string]interface{}{
			"gateway":    "stripe",
			"amount":     10000,
			"currency":   "VND",
			"payment_id": "pay_123456",
		})

	case "email":
		// Simulate email service error
		err := fmt.Errorf("smtp: authentication failed")
		// ⭐ WrapWithMessage() + WithData() + WithCallChain()
		return goerrorkit.WrapWithMessage(err, "Failed to send verification email").
			WithData(map[string]interface{}{
				"to":       "user@example.com",
				"template": "email_verification",
				"smtp":     "smtp.gmail.com:587",
			}).
			WithCallChain()
	}

	return c.JSON(fiberv2.Map{"message": "No error"})
}

// ============================================================================
// Custom Error Handlers - Demonstrate different error types
// ============================================================================

func businessErrorHandler(c *fiberv2.Ctx) error {
	productID := c.Query("product_id", "unknown")

	// Simulate product not found
	if productID == "123" {
		return goerrorkit.NewBusinessError(404, fmt.Sprintf("Product ID=%s not found", productID)).WithData(map[string]interface{}{
			"product_id": productID,
		})
	}

	return c.JSON(fiberv2.Map{
		"message":    "Product available",
		"product_id": productID,
	})
}

func systemErrorHandler(c *fiberv2.Ctx) error {
	// Simulate database connection error
	err := fmt.Errorf("connection refused: database is down")
	return goerrorkit.NewSystemError(err).WithData(map[string]interface{}{
		"database": "postgres",
		"host":     "localhost:5432",
	})
}

func validationErrorHandler(c *fiberv2.Ctx) error {
	age := c.Query("age", "")

	if age == "" {
		return goerrorkit.NewValidationError("Missing parameter 'age'", map[string]interface{}{
			"field":    "age",
			"required": true,
		})
	}

	// Check if age is a number
	var ageInt int
	if _, err := fmt.Sscanf(age, "%d", &ageInt); err != nil {
		return goerrorkit.NewValidationError("Parameter 'age' must be an integer", map[string]interface{}{
			"field":    "age",
			"type":     "integer",
			"received": age,
		})
	}

	if ageInt < 18 {
		return goerrorkit.NewValidationError("Age must be >= 18", map[string]interface{}{
			"field":    "age",
			"min":      18,
			"received": ageInt,
		})
	}

	return c.JSON(fiberv2.Map{
		"message": "Validation successful",
		"age":     ageInt,
	})
}

func authErrorHandler(c *fiberv2.Ctx) error {
	token := c.Get("Authorization")

	// Check if token exists
	if token == "" {
		return goerrorkit.NewAuthError(401, "Unauthorized: Missing authorization token")
	}

	// Simulate invalid token
	if token != "Bearer valid-token-123" {
		return goerrorkit.NewAuthError(401, "Unauthorized: Invalid token").WithData(map[string]interface{}{
			"token_length": len(token),
		})
	}

	// Simulate permission check
	role := c.Get("X-User-Role")
	if role != "admin" {
		return goerrorkit.NewAuthError(403, "Forbidden: Insufficient permissions").WithData(map[string]interface{}{
			"required_role": "admin",
			"user_role":     role,
		})
	}

	return c.JSON(fiberv2.Map{
		"message": "Authentication successful",
		"role":    role,
	})
}

func externalErrorHandler(c *fiberv2.Ctx) error {
	// Simulate external API call failure
	service := c.Query("service", "payment")

	err := fmt.Errorf("timeout after 30s")

	var statusCode int
	var message string

	switch service {
	case "payment":
		statusCode = 502
		message = "Payment gateway not responding"
	case "shipping":
		statusCode = 503
		message = "Shipping service under maintenance"
	case "notification":
		statusCode = 504
		message = "Notification service timeout"
	default:
		statusCode = 502
		message = "External service unavailable"
	}

	return goerrorkit.NewExternalError(statusCode, message, err).WithData(map[string]interface{}{
		"service": service,
		"timeout": "30s",
	})
}

// ============================================================================
// Complex Error Handler - Demonstrate WithCallChain()
// ============================================================================

// complexErrorWithCallChainHandler demonstrates using .WithCallChain()
// to add full call chain to non-panic errors for better debugging
func complexErrorWithCallChainHandler(c *fiberv2.Ctx) error {
	// Simulate a complex operation with multiple function calls
	result, err := processOrder()
	if err != nil {
		return err
	}

	return c.JSON(fiberv2.Map{
		"message": "Order processed",
		"result":  result,
	})
}

func processOrder() (string, error) {
	// Call validation
	if err := validateOrder(); err != nil {
		return "", err
	}

	// Call inventory check
	if err := checkInventory(); err != nil {
		return "", err
	}

	return "success", nil
}

func validateOrder() error {
	// Simulate validation
	isValid := false

	if !isValid {
		// ⭐ Sử dụng .WithCallChain() để thêm full call chain
		// Giúp trace được: complexErrorWithCallChainHandler → processOrder → validateOrder
		return goerrorkit.NewValidationError("Order validation failed", map[string]interface{}{
			"reason": "invalid_order_data",
		}).WithCallChain() // ⭐ Thêm call_chain vào error!
	}

	return nil
}

func checkInventory() error {
	// Simulate inventory check
	stockAvailable := 0

	if stockAvailable == 0 {
		// ⭐ Chain nhiều methods: WithData() + WithCallChain()
		return goerrorkit.NewBusinessError(422, "Insufficient inventory").
			WithData(map[string]interface{}{
				"product_id": "PROD-123",
				"requested":  10,
				"available":  0,
				"warehouse":  "WH-01",
			}).
			WithCallChain() // ⭐ Thêm call_chain để trace flow
	}

	return nil
}
