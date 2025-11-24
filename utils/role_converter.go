package utils

import (
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/models"
)

// RoleConverter provides utilities for converting between role IDs and role names
type RoleConverter struct {
	// GetRoleName function to lookup role name from ID
	GetRoleName func(roleID uint) (string, error)
	// GetRoleID function to lookup role ID from name
	GetRoleID func(roleName string) (uint, error)
}

// RoleIDsToNames converts a slice of role IDs to role names
// Requires a function to lookup role names from IDs
func RoleIDsToNames(roleIDs []uint, getRoleName func(uint) (string, error)) ([]string, error) {
	if len(roleIDs) == 0 {
		return []string{}, nil
	}
	
	names := make([]string, 0, len(roleIDs))
	for _, id := range roleIDs {
		name, err := getRoleName(id)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

// RoleNamesToIDs converts a slice of role names to role IDs
// Requires a function to lookup role IDs from names
func RoleNamesToIDs(roleNames []string, getRoleID func(string) (uint, error)) ([]uint, error) {
	if len(roleNames) == 0 {
		return []uint{}, nil
	}
	
	ids := make([]uint, 0, len(roleNames))
	for _, name := range roleNames {
		id, err := getRoleID(name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ExtractRoleNamesFromRoles extracts role names from a slice of Role models
// This is a pure utility function that doesn't require database access
func ExtractRoleNamesFromRoles(roles []models.Role) []string {
	if len(roles) == 0 {
		return []string{}
	}
	
	names := make([]string, len(roles))
	for i, role := range roles {
		names[i] = role.Name
	}
	return names
}

// ExtractRoleNamesFromRoleInterfaces extracts role names from a slice of RoleInterface
// This is a pure utility function that doesn't require database access
func ExtractRoleNamesFromRoleInterfaces(roles []core.RoleInterface) []string {
	if len(roles) == 0 {
		return []string{}
	}
	
	names := make([]string, len(roles))
	for i, role := range roles {
		names[i] = role.GetName()
	}
	return names
}

// ExtractRoleIDsFromRoles extracts role IDs from a slice of Role models
// This is a pure utility function that doesn't require database access
func ExtractRoleIDsFromRoles(roles []models.Role) []uint {
	if len(roles) == 0 {
		return []uint{}
	}
	
	ids := make([]uint, len(roles))
	for i, role := range roles {
		ids[i] = role.ID
	}
	return ids
}

// ExtractRoleIDsFromRoleInterfaces extracts role IDs from a slice of RoleInterface
// This is a pure utility function that doesn't require database access
func ExtractRoleIDsFromRoleInterfaces(roles []core.RoleInterface) []uint {
	if len(roles) == 0 {
		return []uint{}
	}
	
	ids := make([]uint, len(roles))
	for i, role := range roles {
		ids[i] = role.GetID()
	}
	return ids
}

// ConvertRoleNameMapToIDs converts a map of role name -> role ID to a slice of IDs
// This is useful when you have a map from GetIDsByNames and need a slice
func ConvertRoleNameMapToIDs(nameToIDMap map[string]uint, roleNames []string) []uint {
	if len(roleNames) == 0 {
		return []uint{}
	}
	
	ids := make([]uint, 0, len(roleNames))
	for _, name := range roleNames {
		if id, ok := nameToIDMap[name]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// ConvertRoleIDMapToNames converts a map of role ID -> role name to a slice of names
// This is useful when you have a map and need a slice
func ConvertRoleIDMapToNames(idToNameMap map[uint]string, roleIDs []uint) []string {
	if len(roleIDs) == 0 {
		return []string{}
	}
	
	names := make([]string, 0, len(roleIDs))
	for _, id := range roleIDs {
		if name, ok := idToNameMap[id]; ok {
			names = append(names, name)
		}
	}
	return names
}

