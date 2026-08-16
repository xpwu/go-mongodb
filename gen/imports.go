package gen

import (
	"fmt"
	"path"
	"sort"
)

// allImports 管理生成文件所需的所有 import
type allImports struct {
	data  map[string]string
	alias aliasNames
	exc   map[string]bool
}

// aliasNames 已使用的别名集合
type aliasNames map[string]bool

func (a aliasNames) get(expect string) string {
	test := expect
	num := 1
	for a[test] {
		num++
		test = fmt.Sprintf("%s%d", expect, num)
	}
	a[test] = true
	return test
}

// newAllImports 创建空的 import 管理器
func newAllImports() *allImports {
	return &allImports{
		data:  make(map[string]string),
		alias: aliasNames{},
		exc:   make(map[string]bool),
	}
}

// exclude 排除指定包路径（不生成 import）
func (m *allImports) exclude(paths string) {
	m.exc[paths] = true
}

// add 添加包路径，返回别名
func (m *allImports) add(paths string) (alias string) {
	if paths == "" || m.exc[paths] {
		return ""
	}
	if a, ok := m.data[paths]; ok {
		return a
	}
	a := m.alias.get(path.Base(paths))
	m.data[paths] = a
	return a
}

// addDot 给非空字符串加 "." 后缀
func addDot(s string) string {
	if len(s) == 0 {
		return s
	}
	return s + "."
}

// importTemp 单个 import 声明
type importTemp struct {
	Alias  string
	Import string
}

// all 返回排序后的 import 列表
func (m *allImports) all() []importTemp {
	ret := make([]importTemp, len(m.data))
	var keys []string
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for j, k := range keys {
		ret[j] = importTemp{
			Alias:  m.data[k],
			Import: k,
		}
	}
	return ret
}
