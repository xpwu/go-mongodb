package base

import "github.com/xpwu/go-mongodb/zdemo/elsetype"

type Order struct {
	Uint64                    uint64
	String                    string
	LikeInt_elsetype_TimeType elsetype.TimeType
	LikeUint_elsetype_UTime   elsetype.UTime
}
