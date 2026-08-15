package gen

import (
	"fmt"
	"path"
	"sort"
)

type allImports struct {
	data  map[string]string
	alias aliasNames
	exc   map[string]bool
}

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

func newAllImports() *allImports {
	return &allImports{
		data:  make(map[string]string),
		alias: aliasNames{},
		exc:   make(map[string]bool),
	}
}

func (m *allImports) exclude(paths string) {
	m.exc[paths] = true
}

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

func addDot(s string) string {
	if len(s) == 0 {
		return s
	}
	return s + "."
}

type importTemp struct {
	Alias  string
	Import string
}

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
