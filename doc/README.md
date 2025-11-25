# Tài liệu Kiến trúc AuthKit

Tài liệu này mô tả kiến trúc của AuthKit theo nguyên tắc từ tổng quát đến chi tiết, từ dễ đến khó. Mỗi tài liệu bao gồm sơ đồ mermaid, ví dụ code chi tiết, và giải thích kỹ thuật đầy đủ.

---

## Mục lục

### 1. [Tổng quan về AuthKit](./01-tong-quan.md)
Tài liệu giới thiệu tổng quan về AuthKit với các sơ đồ kiến trúc và luồng xử lý:
- **AuthKit là gì?**: Module Go tái sử dụng cao cho Fiber REST API
- **Mục đích và phạm vi**: Authentication & Authorization hoàn chỉnh
- **Các tính năng chính**: 
  - JWT-based Authentication với sequence diagrams
  - Rule-based Authorization với flowchart chi tiết
  - Role Management và Route Management
  - Generic Types và Extensibility
- **Kiến trúc tổng thể**: Sơ đồ high-level với 5 lớp kiến trúc

### 2. [Kiến trúc tổng thể](./02-kien-truc-tong-the.md)
Tài liệu chi tiết về thiết kế và cách AuthKit hoạt động:
- **Mô hình kiến trúc**: Layered Architecture với 5 lớp (Router, Middleware, Handler, Service, Repository)
- **Luồng xử lý request**: Sequence diagram từ client đến database và response
- **Các thành phần chính**: 
  - AuthKit container với dependency graph
  - RouteRegistry với exact map và pattern list
  - AuthKitBuilder với Builder Pattern
  - Route Registration Flow chi tiết
- **Middleware Flow**: Flowchart chi tiết authentication và authorization

### 3. [Middleware và Security](./03-middleware-security.md)
Tài liệu về cách AuthKit bảo vệ ứng dụng:
- **Authentication Middleware**: 
  - Sequence diagram chi tiết luồng xử lý
  - JWT token extraction và validation
  - Bảo mật Role IDs trong token
- **Authorization Middleware**: 
  - Flowchart đầy đủ rule evaluation
  - Rule matching algorithm (exact + pattern)
  - Early exit patterns (PUBLIC, super_admin)
- **Cơ chế cache**: 
  - Rules cache structure với nested maps
  - Cache refresh strategy
  - Thread safety với RWMutex

### 4. [Hệ thống phân quyền](./04-he-thong-phan-quyen.md)
Tài liệu chi tiết về hệ thống phân quyền rule-based:
- **Rule-based Authorization**: 
  - Rule Model và Rule Matching Algorithm với flowchart
  - Multiple rules cho cùng endpoint
- **Các loại Access Type**: 
  - PUBLIC: Sequence diagram cho anonymous users
  - ALLOW: Flowchart với role checking
  - FORBID: Flowchart với priority evaluation
- **Role và User-Role Relationship**: 
  - ER diagram chi tiết
  - super_admin role đặc biệt với bypass logic
- **Route Sync và Rule Management**: 
  - Sequence diagram sync routes flow
  - Fixed rules, Override rules và Non-fixed rules
  - Rule Management API với ví dụ curl

### 5. [Database Schema và Models](./05-database-schema-models.md)
Tài liệu về database schema, migrations và seeding:
- **ER Diagram**: Sơ đồ quan hệ đầy đủ giữa các bảng
- **Chi tiết từng bảng**: 
  - Users Table với custom fields
  - Roles Table với system roles
  - User_Roles Table (Many-to-Many) với cascade delete
  - Rules Table với PostgreSQL integer[] array
- **Cơ chế Migration**: 
  - Up/Down migration flow với sequence diagram
  - Schema migrations table tracking
- **Cơ chế Seeding**: 
  - Seeding flow với sequence diagram
  - Upsert pattern cho roles và users
- **Cơ chế Upsert**: 
  - FirstOrCreate vs Check then Create
  - Best practices và so sánh các patterns

### 6. [Generic Types và Extensibility](./06-generic-types-extensibility.md)
Tài liệu về Generic Types và cách mở rộng AuthKit:
- **Tại sao cần Generic Types**: 
  - So sánh với cách không dùng Generic (phải ép kiểu)
  - Lợi ích về type safety và code reuse
- **Generic Design Pattern**: 
  - Class diagram Generic Types Architecture
  - Type Constraints và Benefits
- **Dependency Injection và Builder Pattern**: 
  - AuthKitBuilder với sequence diagram khởi tạo
  - Dependency Injection Flow chi tiết
- **UserInterface và RoleInterface**: 
  - Class diagram interface implementation
  - Implementation với BaseUser và BaseRole
- **Custom User Model**: 
  - Ví dụ đầy đủ CustomUser với Mobile và Address
  - So sánh không dùng Generic vs dùng Generic
- **Custom Role Model**: 
  - Ví dụ CustomRole với Department và Level
  - Tổng hợp sử dụng cả CustomUser và CustomRole

### 7. [Cơ chế hoạt động chi tiết](./07-co-che-hoat-dong-chi-tiet.md)
Tài liệu về implementation details ở mức code:
- **JWT Token Implementation**: 
  - Claims structure với RoleIDs protection
  - Token generation và validation process
  - Algorithm confusion prevention
- **Password Hashing Implementation**: 
  - Bcrypt hashing với cost factor
  - Password verification process
- **Rule Matching Algorithm Implementation**: 
  - Cache data structures (exact map + nested maps)
  - Finding matching rules với O(1) exact và optimized pattern
  - Segment counting và path pattern matching algorithms
- **Cache Refresh Implementation**: 
  - Cache refresh process với thread safety
  - Cache invalidation strategy
- **User ID Generation**: 
  - Cryptographically secure random generation
  - Collision probability analysis

### 8. [Tích hợp và Sử dụng](./08-tich-hop-su-dung.md)
Hướng dẫn thực tế để tích hợp AuthKit:
- **Quick Start**: 
  - Ví dụ đầy đủ với code
  - Checklist tích hợp từng bước
- **Common Use Cases**: 
  - Sử dụng với Custom User Model
  - Định nghĩa routes với các Access Types
  - Lấy User từ Context (type-safe)
  - Sử dụng AuthService trực tiếp
- **Error Handling**: 
  - goerrorkit error types
  - Ví dụ xử lý errors
- **Troubleshooting**: 
  - Routes không được sync
  - Token không hợp lệ
  - User không có quyền truy cập
  - Custom fields không được lưu
  - Role names không được convert
- **Best Practices**: Do's và Don'ts

### 9. [Tối ưu hóa và Best Practices](./09-toi-uu-hoa-best-practices.md)
Tài liệu về tối ưu hóa performance và best practices:
- **Authorization Flow - Hot Path Analysis**: 
  - Performance critical points
  - Early exit patterns
- **Tối ưu Rule Matching**: 
  - Exact Match O(1) lookup
  - Pattern Matching với nested maps optimization
  - Segment Counting zero allocation
- **Tối ưu Role Checking**: 
  - Role IDs từ JWT token (zero DB query)
  - Role ID Map O(1) lookup
  - super_admin Cache O(1) check
- **Early Exit Patterns**: 
  - Early exit order (từ nhanh nhất đến chậm nhất)
  - Rule evaluation priority
- **Cache Management**: 
  - Cache structure optimization
  - Thread safety với RWMutex
  - Cache invalidation strategy
- **Performance Benchmarks**: 
  - Typical performance per request
  - Throughput impact (optimized vs unoptimized)
- **Best Practices Summary**: Do's và Don'ts

### 10. [Kiến trúc Microservice với AuthKit](./10-microservice_auth.md)
So sánh chi tiết hai phương án triển khai authentication và authorization trong kiến trúc microservice:
- **Tổng quan**: Bài toán và hai phương án (Direct DB Connection vs Auth Service API)
- **Phương án 1 - Direct DB Connection**: 
  - Kiến trúc chi tiết với sơ đồ
  - Luồng xử lý login và request
  - Implementation code đầy đủ
- **Phương án 2 - Auth Service API**: 
  - Kiến trúc tập trung với sơ đồ
  - Luồng xử lý qua HTTP API
  - Implementation với HTTP client và custom middleware
- **So sánh chi tiết**: 
  - Tốc độ xử lý (latency)
  - Throughput (số lượng request lớn)
  - Dễ code và dễ bảo trì
  - Bảo mật
- **Khuyến nghị**: Khi nào nên dùng phương án nào
- **Với codebase hiện tại**: Phương án nào dễ triển khai hơn

### 11. [Unit Tests cho Authorization Middleware](./11_authorization_unittest.md)
Tài liệu về unit tests cho authorization middleware không cần kết nối PostgreSQL:
- **Cách chạy tests**: Các lệnh test và coverage
- **Các hàm đã được test**: 
  - Pure Functions (countSegments, matchPath)
  - Cache và Rule Matching (findMatchingRules, refreshCache)
  - Authorization Logic với Roles (Authorize middleware)
- **Cách test hoạt động**: Kỹ thuật test không cần database
- **Test Coverage**: Coverage hiện tại và lý do
- **Mở rộng Tests**: Hướng dẫn tạo integration tests với database

### 12. [Unit Tests cho Authorization Middleware Integration](./12_authorization_integration_test.md)
Tài liệu về integration tests cho authorization middleware với kết nối PostgreSQL:
- **Integration Tests**: Test với database thực sự
- **Test Scenarios**: Các test cases với real repositories và database
- **Setup và Teardown**: Cách setup test database và cleanup
- **Best Practices**: Do's và Don'ts cho integration tests

---

## Cách đọc tài liệu

### Người mới bắt đầu
1. Bắt đầu từ **[Tổng quan về AuthKit](./01-tong-quan.md)** để hiểu AuthKit là gì và các tính năng chính
2. Đọc **[Kiến trúc tổng thể](./02-kien-truc-tong-the.md)** để hiểu cách AuthKit được thiết kế
3. Xem **[Tích hợp và Sử dụng](./08-tich-hop-su-dung.md)** để bắt đầu tích hợp vào dự án

### Tìm hiểu kiến trúc
- **[Kiến trúc tổng thể](./02-kien-truc-tong-the.md)**: Layered Architecture, Request Flow, Components
- **[Middleware và Security](./03-middleware-security.md)**: Authentication và Authorization flow chi tiết
- **[Hệ thống phân quyền](./04-he-thong-phan-quyen.md)**: Rule-based authorization và role management

### Tích hợp vào dự án
- **[Tích hợp và Sử dụng](./08-tich-hop-su-dung.md)**: Quick start, use cases, troubleshooting
- **[Database Schema và Models](./05-database-schema-models.md)**: Migrations, seeding, upsert patterns

### Tùy chỉnh và mở rộng
- **[Generic Types và Extensibility](./06-generic-types-extensibility.md)**: Custom User/Role models với type safety
- **[Cơ chế hoạt động chi tiết](./07-co-che-hoat-dong-chi-tiet.md)**: Implementation details nếu cần customize

### Tối ưu hóa
- **[Tối ưu hóa và Best Practices](./09-toi-uu-hoa-best-practices.md)**: Performance optimizations và benchmarks

### Kiến trúc Microservice
- **[Kiến trúc Microservice với AuthKit](./10-microservice_auth.md)**: So sánh hai phương án triển khai SSO trong microservice

### Testing
- **[Unit Tests cho Authorization Middleware](./11_authorization_unittest.md)**: Unit tests không cần database

---

## Đặc điểm tài liệu

Mỗi tài liệu bao gồm:
- ✅ **Sơ đồ Mermaid**: Sequence diagrams, flowcharts, class diagrams, ER diagrams
- ✅ **Ví dụ code**: Code examples đầy đủ và thực tế
- ✅ **Giải thích chi tiết**: Từng bước, từng component được giải thích rõ ràng
- ✅ **Best Practices**: Do's và Don'ts cho từng chủ đề
- ✅ **Troubleshooting**: Các vấn đề thường gặp và cách giải quyết

---

## File gốc

File gốc `Architect.md` đã được chia thành 9 file riêng biệt để dễ đọc và quản lý. Mỗi file tập trung vào một chủ đề cụ thể với nội dung chi tiết, sơ đồ và ví dụ code đầy đủ.

File `microservice_auth.md` được thêm vào để hướng dẫn triển khai AuthKit trong kiến trúc microservice với so sánh chi tiết giữa các phương án.

