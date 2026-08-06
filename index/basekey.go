package index

type BaseKey interface {
	AscIndex(opts ...Option) Key
	DescIndex(opts ...Option) Key
}
