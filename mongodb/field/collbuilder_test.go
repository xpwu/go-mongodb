package field

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestImports(t *testing.T) {
	a := assert.New(t)

	imps := newAllImports()

	alia := imps.add("abc/efg/field")
	a.Equal("field", alia)

	alia = imps.add("abc/efg/field")
	a.Equal("field", alia)
}
