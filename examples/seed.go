package main

import (
	"fmt"

	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// SeedData seeds initial data (roles and users) into the database
// Note: Rules sẽ được sync tự động từ routes trong main.go
func SeedData(db *gorm.DB) error {
	// Initialize roles
	if err := initRoles(db); err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to initialize roles").
			WithData(map[string]interface{}{
				"operation": "init_roles",
			})
	}

	// Initialize users
	// Note: Rules sẽ được sync tự động từ routes trong main.go
	if err := initUsers(db); err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to initialize users").
			WithData(map[string]interface{}{
				"operation": "init_users",
			})
	}

	return nil
}

// initRoles initializes default roles using UPSERT
func initRoles(db *gorm.DB) error {
	roles := []string{"admin", "editor", "author", "reader"}

	for _, roleName := range roles {
		role := &authkit.Role{
			Name: roleName,
		}

		// FirstOrCreate: tìm theo Name, nếu không có thì tạo mới
		result := db.Where("name = ?", roleName).FirstOrCreate(role)
		if result.Error != nil {
			return goerrorkit.WrapWithMessage(result.Error, fmt.Sprintf("Failed to initialize role %s", roleName)).
				WithData(map[string]interface{}{
					"role_name": roleName,
				})
		}

		// result.RowsAffected > 0 nghĩa là đã tạo mới
		if result.RowsAffected > 0 {
			fmt.Printf("Created role: %s\n", roleName)
		}
	}

	return nil
}

// initUsers initializes default test users with roles using UPSERT
func initUsers(db *gorm.DB) error {
	// Define test users
	testUsers := []struct {
		email    string
		password string
		fullName string
		roles    []string
	}{
		{
			email:    "admin@gmail.com",
			password: "123456",
			fullName: "Admin User",
			roles:    []string{"admin"},
		},
		{
			email:    "editor@gmail.com",
			password: "123456",
			fullName: "Editor User",
			roles:    []string{"editor", "author"},
		},
		{
			email:    "author1@gmail.com",
			password: "123456",
			fullName: "Author 1",
			roles:    []string{"author"},
		},
		{
			email:    "author2@gmail.com",
			password: "123456",
			fullName: "Author 2",
			roles:    []string{"author"},
		},
		{
			email:    "reader@gmail.com",
			password: "123456",
			fullName: "Reader User",
			roles:    []string{"reader"},
		},
	}

	for _, userData := range testUsers {
		// Hash password
		hashedPassword, err := utils.HashPassword(userData.password)
		if err != nil {
			return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to hash password for user %s", userData.email)).
				WithData(map[string]interface{}{
					"email": userData.email,
				})
		}

		// Create or get user
		user := &authkit.User{
			Email:    userData.email,
			Password: hashedPassword,
			FullName: userData.fullName,
			Active:   true,
		}

		// FirstOrCreate: tìm theo Email, nếu không có thì tạo mới
		result := db.Where("email = ?", userData.email).FirstOrCreate(user)
		if result.Error != nil {
			return goerrorkit.WrapWithMessage(result.Error, fmt.Sprintf("Failed to initialize user %s", userData.email)).
				WithData(map[string]interface{}{
					"email": userData.email,
				})
		}

		// Update password if user already exists (in case password changed)
		if result.RowsAffected == 0 {
			user.Password = hashedPassword
			if err := db.Save(user).Error; err != nil {
				return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to update password for user %s", userData.email)).
					WithData(map[string]interface{}{
						"email": userData.email,
					})
			}
		} else {
			fmt.Printf("Created user: %s\n", userData.email)
		}

		// Assign roles to user
		var roles []authkit.Role
		for _, roleName := range userData.roles {
			var role authkit.Role
			if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
				return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to find role %s for user %s", roleName, userData.email)).
					WithData(map[string]interface{}{
						"email":     userData.email,
						"role_name": roleName,
					})
			}
			roles = append(roles, role)
		}

		// Replace all roles for the user
		if err := db.Model(user).Association("Roles").Replace(roles); err != nil {
			return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to assign roles to user %s", userData.email)).
				WithData(map[string]interface{}{
					"email": userData.email,
					"roles": userData.roles,
				})
		}
	}

	return nil
}
