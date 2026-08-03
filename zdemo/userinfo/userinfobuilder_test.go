package userinfo

import (
	"fmt"
	"github.com/xpwu/go-mongodb/fields"
	"github.com/xpwu/go-mongodb/zdemo/elsetype"
)

func ExampleStructFieldBuilder() {

	builder := fields.NewStructFieldBuilder()

	fields.BuildColl[UserInfo](builder)
	fields.BuildColl[elsetype.ThirdParty](builder)

	fmt.Println(true)
	// Output:
	// true
}
