package newfield

import (
	"fmt"
	"github.com/xpwu/go-db-mongo/mongodb/field"
	"github.com/xpwu/go-db-mongo/mongodb/field/elsejson"
)

func ExampleCollBuilder() {

	builder := field.NewCollBuilder()

	field.BuildColl[UserInfo](builder)
	field.BuildColl[elsejson.ThirdParty](builder)

	fmt.Println(true)
	// Output:
	// true
}
