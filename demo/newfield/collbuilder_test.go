package newfield

import (
	"fmt"
	"github.com/xpwu/go-mongodb/demo/elsejson"
	"github.com/xpwu/go-mongodb/fields"
)

func ExampleCollBuilder() {

	builder := fields.NewCollBuilder()

	fields.BuildColl[UserInfo](builder)
	fields.BuildColl[elsejson.ThirdParty](builder)

	fmt.Println(true)
	// Output:
	// true
}
