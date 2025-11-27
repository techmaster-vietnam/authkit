package utils

import "reflect"

// NewGenericInstance tạo instance mới của generic type T, xử lý cả pointer và value types
// Helper function tổng quát để tái sử dụng khi cần tạo instance của generic type
// 
// Ví dụ sử dụng:
//   role, rv := utils.NewGenericInstance[*BaseRole]()
//   rv.FieldByName("ID").SetUint(1)
//   rv.FieldByName("Name").SetString("admin")
//
//   user, rv := utils.NewGenericInstance[*CustomUser]()
//   rv.FieldByName("Email").SetString("user@example.com")
//
// Returns:
//   - instance: Instance mới của type T (đã được khởi tạo nếu là pointer type)
//   - rv: reflect.Value của struct để có thể set fields (luôn là struct, không phải pointer)
func NewGenericInstance[T any]() (T, reflect.Value) {
	var instance T
	instanceValue := reflect.ValueOf(instance)
	
	// Nếu T là pointer type (như *BaseRole, *CustomUser), cần tạo instance mới
	if instanceValue.Kind() == reflect.Ptr && instanceValue.IsNil() {
		elemType := instanceValue.Type().Elem()
		newPtr := reflect.New(elemType)
		instance = newPtr.Interface().(T)
		instanceValue = newPtr
	}
	
	// Lấy reflect.Value của struct để set fields
	// Nếu instance là pointer, lấy element; nếu không, lấy address rồi element
	var rv reflect.Value
	if instanceValue.Kind() == reflect.Ptr {
		rv = instanceValue.Elem()
	} else {
		rv = reflect.ValueOf(&instance).Elem()
	}
	
	return instance, rv
}

