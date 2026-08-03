package x

import (
	"reflect"
	"strings"
)

func TypeFor[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func BaseTypeName(t reflect.Type) string {
	name := t.Name()
	// 去掉泛型参数部分（如果有）
	if idx := strings.Index(name, "["); idx != -1 {
		return name[:idx]
	}
	return name
}

// SanitizePackageName 从路径中提取最后一段，并转换为合法的 Go 包名
func SanitizePackageName(path string) string {
	// 1. 获取 '/' 分隔的最后一段
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		path = path[idx+1:]
	}

	// 2. 替换非 Go 包名合法字符（字母、数字、下划线）为 '_'
	var builder strings.Builder
	for _, r := range path {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}
	result := builder.String()

	// 3. 如果结果为空
	if result == "" {
		return result
	}

	// 4. 如果首字符是数字，替换为_
	if result[0] >= '0' && result[0] <= '9' {
		result = "_" + result[1:]
	}

	return result
}
