package userinfo

import (
	"fmt"
	"github.com/xpwu/go-mongodb/fields"
	"github.com/xpwu/go-mongodb/zdemo/elsetype"
	"reflect"
)

func ExampleStructFieldBuilder() {

	builder := fields.NewStructFieldBuilder()

	fields.BuildStruct[UserInfo](builder)
	builder.Build(reflect.TypeOf(elsetype.ThirdParty{}))

	fmt.Println(true)
	// Output:
	// true
}
