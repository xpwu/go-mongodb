package index

type BaseKey interface {
	AscIndex() Key
	DescIndex() Key
}
