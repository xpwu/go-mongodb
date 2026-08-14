package base

import "github.com/xpwu/go-mongodb/zdemo/elsetype"

type Order struct {
	Time      uint64
	Processor string
	Age       elsetype.TimeType
	Hour      elsetype.UTime
}
