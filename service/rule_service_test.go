package service

import (
	"reflect"
	"testing"

	"github.com/techmaster-vietnam/authkit/repository"
)

// Lưu ý: RuleService hiện tại sử dụng concrete types (*repository.RuleRepository và *repository.RoleRepository)
// thay vì interface, nên không thể mock trực tiếp như BaseRoleService.
// Để test đầy đủ các methods (UpdateRule, ListRules, GetByID), cần:
// 1. Refactor RuleService để sử dụng interface cho repositories (giống BaseRoleService)
// 2. Hoặc sử dụng test database thật để test
// Xem base_role_service_test.go để tham khảo cách test với interface-based repositories

func TestRuleService_NewRuleService(t *testing.T) {
	ruleRepo := repository.NewRuleRepository(nil, "")
	roleRepo := repository.NewRoleRepository(nil)

	service := NewRuleService(ruleRepo, roleRepo)

	if service == nil {
		t.Errorf("Expected service to be created, got nil")
	}
}

func TestRuleService_SetCacheInvalidator(t *testing.T) {
	ruleRepo := repository.NewRuleRepository(nil, "")
	roleRepo := repository.NewRoleRepository(nil)
	mockCacheInvalidator := NewMockCacheInvalidator()

	service := NewRuleService(ruleRepo, roleRepo)

	// Kiểm tra ban đầu không có cache invalidator
	serviceValue := reflect.ValueOf(service).Elem()
	cacheInvalidatorField := serviceValue.FieldByName("cacheInvalidator")
	if !cacheInvalidatorField.IsNil() {
		t.Errorf("Expected cacheInvalidator to be nil initially")
	}

	// Set cache invalidator
	service.SetCacheInvalidator(mockCacheInvalidator)

	// Kiểm tra cache invalidator đã được set bằng cách kiểm tra không nil
	cacheInvalidatorField = serviceValue.FieldByName("cacheInvalidator")
	if cacheInvalidatorField.IsNil() {
		t.Errorf("Expected cacheInvalidator to be set, got nil")
	}

	// Kiểm tra cache invalidator đã được gọi khi cần (sẽ test trong test UpdateRule)
	// Ở đây chỉ đảm bảo SetCacheInvalidator không panic
}

// TestRuleService_UpdateRule tests UpdateRule method
// Lưu ý: Test này cần test database hoặc refactor RuleService để sử dụng interface
// Vì RuleService sử dụng concrete types, không thể mock trực tiếp
func TestRuleService_UpdateRule(t *testing.T) {
	// Vì RuleService sử dụng concrete repository types,
	// chúng ta không thể mock trực tiếp như BaseRoleService
	// Test này sẽ cần một test database hoặc refactor để sử dụng interface

	// Tạm thời skip test này và ghi chú rằng cần refactor
	t.Skip("Test này cần test database hoặc refactor RuleService để sử dụng interface cho repositories. " +
		"Xem base_role_service_test.go để tham khảo cách test với interface-based repositories.")
}

func TestRuleService_ListRules(t *testing.T) {
	t.Skip("Test này cần test database hoặc refactor RuleService để sử dụng interface cho repositories")
}

func TestRuleService_GetByID(t *testing.T) {
	t.Skip("Test này cần test database hoặc refactor RuleService để sử dụng interface cho repositories")
}
