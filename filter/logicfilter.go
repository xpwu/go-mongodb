package filter

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type logic struct {
	operator string
	filters  []Filter
}

func (l *logic) ToBsonD() *bson.D {

	var value []*bson.D

	for _, filter := range l.filters {
		b, ok := filter.(*logic)
		if !ok || b.operator != l.operator {
			value = append(value, filter.ToBsonD())
			continue
		}

		// merge operator
		for _, filter := range b.filters {
			value = append(value, filter.ToBsonD())
		}
	}

	return &bson.D{{l.operator, value}}
}

const (
	or  = `$or`
	and = `$and`
)

func newLogic(operator string, filter1, filter2 Filter, filters ...Filter) *logic {
	f := make([]Filter, 2, 2+len(filters))
	f[0] = filter1
	f[1] = filter2
	f = append(f, filters...)

	return &logic{
		operator: operator,
		filters:  f,
	}
}

type nor struct {
	filters []Filter
}

func (l *nor) ToBsonD() *bson.D {

	var value []*bson.D

	for _, filter := range l.filters {
		value = append(value, filter.ToBsonD())
	}

	return &bson.D{{`$nor`, value}}
}

func newNor(filter1, filter2 Filter, filters ...Filter) *nor {
	f := make([]Filter, 2, 2+len(filters))
	f[0] = filter1
	f[1] = filter2
	f = append(f, filters...)

	return &nor{
		filters: f,
	}
}
