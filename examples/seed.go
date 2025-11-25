package main

import (
	"fmt"
	"os"

	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedData seeds initial data (roles and users) into the database
// Note: Rules sẽ được sync tự động từ routes trong main.go
func SeedData(db *gorm.DB) error {
	// Initialize roles
	if err := initRoles(db); err != nil {
		goerrorkit.LogError(goerrorkit.NewSystemError(err), "Failed to initialize roles")
		return err
	}

	// Initialize users
	// Note: Rules sẽ được sync tự động từ routes trong main.go
	if err := initUsers(db); err != nil {
		goerrorkit.LogError(goerrorkit.NewSystemError(err), "Failed to initialize users")
		return err
	}

	return nil
}

// initRoles initializes default roles using UPSERT
func initRoles(db *gorm.DB) error {
	roles := []struct {
		id     uint
		name   string
		system bool // Thêm field này
	}{
		{id: 1, name: "super_admin", system: true},
		{id: 2, name: "admin"},
		{id: 3, name: "editor"},
		{id: 4, name: "author"},
		{id: 5, name: "reader"},
	}

	for _, roleData := range roles {
		role := &authkit.Role{
			ID:     roleData.id,
			Name:   roleData.name,
			System: roleData.system, // Set system flag
		}

		// Sử dụng UPSERT của PostgreSQL: INSERT ... ON CONFLICT ... DO UPDATE
		// Nếu role đã tồn tại (conflict trên name), cập nhật System flag
		// Lưu ý: tên cột trong database là "is_system" (không phải "system")
		result := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"is_system"}),
		}).Create(role)

		if result.Error != nil {
			goerrorkit.LogError(goerrorkit.NewSystemError(result.Error), fmt.Sprintf("Failed to initialize role %s", roleData.name))
			return result.Error
		}

		// result.RowsAffected > 0 nghĩa là đã tạo mới hoặc cập nhật
		if result.RowsAffected > 0 {
			fmt.Printf("Upserted role: %s (ID: %d, System: %v)\n", roleData.name, roleData.id, roleData.system)
		}
	}

	return nil
}

// initUsers initializes default test users with roles using UPSERT
func initUsers(db *gorm.DB) error {
	// Đọc password từ environment variable
	superAdminPassword := os.Getenv("SUPER_ADMIN_PASSWORD")
	if superAdminPassword == "" {
		// Nếu không có env var, bỏ qua việc tạo super_admin
		fmt.Println("Warning: SUPER_ADMIN_PASSWORD not set, skipping super_admin user creation")
	}

	// Define test users với custom fields mobile và address
	testUsers := []struct {
		email    string
		password string
		fullName string
		mobile   string
		address  string
		roles    []string
	}{
		{
			email:    "cuong@techmaster.vn",
			password: superAdminPassword,
			fullName: "Super Admin User",
			mobile:   "0902209011",
			address:  "14 ngõ 4 Nguyễn Đình Chiểu, Hà nội",
			roles:    []string{"super_admin"},
		},
		{
			email:    "admin@gmail.com",
			password: "123456",
			fullName: "Admin User",
			mobile:   "0901234567",
			address:  "123 Admin Street, Ho Chi Minh City",
			roles:    []string{"admin"},
		},
		{
			email:    "editor@gmail.com",
			password: "123456",
			fullName: "Editor User",
			mobile:   "0902345678",
			address:  "456 Editor Avenue, Hanoi",
			roles:    []string{"editor", "author"},
		},
		{
			email:    "author1@gmail.com",
			password: "123456",
			fullName: "Author 1",
			mobile:   "0903456789",
			address:  "789 Author Road, Da Nang",
			roles:    []string{"author"},
		},
		{
			email:    "author2@gmail.com",
			password: "123456",
			fullName: "Author 2",
			mobile:   "0904567890",
			address:  "321 Writer Lane, Can Tho",
			roles:    []string{"author"},
		},
		{
			email:    "reader@gmail.com",
			password: "123456",
			fullName: "Reader User",
			mobile:   "0905678901",
			address:  "654 Reader Boulevard, Hai Phong",
			roles:    []string{"reader"},
		},
		{
			email:    "bob@gmail.com",
			password: "123456",
			fullName: "Bob",
			mobile:   "0906789012",
			address:  "987 Bob Street, Ho Chi Minh City",
			roles:    []string{"reader", "author", "editor"},
		},
	}

	for _, userData := range testUsers {
		// Skip super_admin user nếu không có password
		if userData.password == "" && contains(userData.roles, "super_admin") {
			fmt.Printf("Skipping user %s: password not provided\n", userData.email)
			continue
		}

		// Hash password
		hashedPassword, err := utils.HashPassword(userData.password)
		if err != nil {
			goerrorkit.LogError(goerrorkit.NewSystemError(err), fmt.Sprintf("Failed to hash password for user %s", userData.email))
			return err
		}

		// Tạo user với UPSERT của PostgreSQL: INSERT ... ON CONFLICT ... DO UPDATE
		// Nếu user đã tồn tại (conflict trên email), cập nhật các trường password, full_name, mobile, address, is_active, deleted_at
		// Lưu ý: tên cột trong database là "is_active" (không phải "active")
		// deleted_at sẽ được set thành NULL để khôi phục user nếu đã bị soft delete
		user := &CustomUser{
			BaseUser: authkit.BaseUser{
				Email:    userData.email,
				Password: hashedPassword,
				FullName: userData.fullName,
				Active:   true,
			},
			Mobile:  userData.mobile,
			Address: userData.address,
		}

		// Sử dụng UPSERT: nếu email đã tồn tại thì update, nếu chưa thì insert
		result := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}},
			DoUpdates: clause.AssignmentColumns([]string{"password", "full_name", "mobile", "address", "is_active", "deleted_at"}),
		}).Create(user)

		if result.Error != nil {
			goerrorkit.LogError(goerrorkit.NewSystemError(result.Error), fmt.Sprintf("Failed to upsert user %s", userData.email))
			return result.Error
		}

		if result.RowsAffected > 0 {
			fmt.Printf("Upserted user: %s\n", userData.email)
		}

		// Load lại user để đảm bảo có ID đúng (cần thiết cho việc gán roles)
		// Tạo struct mới để tránh GORM sử dụng ID cũ trong struct user
		loadedUser := &CustomUser{}
		if err := db.Where("email = ?", userData.email).First(loadedUser).Error; err != nil {
			goerrorkit.LogError(goerrorkit.NewSystemError(err), fmt.Sprintf("Failed to load user %s after upsert", userData.email))
			return err
		}
		user = loadedUser

		// Assign roles to user
		var roles []authkit.Role
		for _, roleName := range userData.roles {
			var role authkit.Role
			if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
				goerrorkit.LogError(goerrorkit.NewSystemError(err), fmt.Sprintf("Failed to find role %s for user %s", roleName, userData.email))
				return err
			}
			roles = append(roles, role)
		}

		// Replace all roles for the user
		if err := db.Model(user).Association("Roles").Replace(roles); err != nil {
			goerrorkit.LogError(goerrorkit.NewSystemError(err), fmt.Sprintf("Failed to assign roles to user %s", userData.email))
			return err
		}
	}

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
