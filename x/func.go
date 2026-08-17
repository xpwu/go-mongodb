package x

import (
	"crypto/sha256"
	"encoding/base64"
	"go.mongodb.org/mongo-driver/v2/bson"
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

func BaseTypeNameFromName(name string) string {
	// 去掉泛型参数部分（如果有）
	if idx := strings.Index(name, "["); idx != -1 {
		return name[:idx]
	}
	return name
}

func LastSubPath(path string) string {
	// 1. 获取 '/' 分隔的最后一段
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		path = path[idx+1:]
	}

	return path
}

// SanitizePackageName 并转换路径为合法的 Go 包名
func SanitizePackageName(path string) string {
	// 1. 替换非 Go 包名合法字符（字母、数字、下划线）为 '_'
	var builder strings.Builder
	for _, r := range path {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}
	result := builder.String()

	// 2. 如果结果为空
	if result == "" {
		return result
	}

	// 3. 如果首字符是数字，替换为_
	if result[0] >= '0' && result[0] <= '9' {
		result = "_" + result[1:]
	}

	return result
}

func CapitalizeASCII(s string) string {
	if s == "" {
		return ""
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return string(b)
}

func ToBsonA[T any](docs []T) bson.A {
	a := make(bson.A, len(docs))
	for i, d := range docs {
		a[i] = d
	}
	return a
}

func DtoM(doc bson.D) bson.M {
	ret := bson.M{}
	for _, e := range doc {
		ret[e.Key] = e.Value
	}

	return ret
}

func DtoMDeeply(doc bson.D) bson.M {
	ret := bson.M{}
	for _, e := range doc {
		if ed, ok := e.Value.(bson.D); ok {
			ret[e.Key] = DtoMDeeply(ed)
		} else {
			ret[e.Key] = e.Value
		}
	}

	return ret
}

func MtoDDeeply(m bson.M) bson.D {
	ret := bson.D{}
	for k, v := range m {
		switch vv := v.(type) {
		case bson.M:
			ret = append(ret, bson.E{Key: k, Value: MtoDDeeply(vv)})
		case bson.A:
			r := bson.A{}
			for _, a := range vv {
				if am, ok := a.(bson.M); ok {
					r = append(r, MtoDDeeply(am))
				} else {
					r = append(r, a)
				}
			}
			ret = append(ret, bson.E{Key: k, Value: r})
		default:
			ret = append(ret, bson.E{Key: k, Value: v})
		}
	}

	return ret
}

func Base6408(s string) string {
	sha256v := sha256.Sum256([]byte(s))
	r := base64.StdEncoding.EncodeToString(sha256v[:])
	return r[0:8]
}
