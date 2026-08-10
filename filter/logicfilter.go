package filter

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"strings"
)

// filter: <field>: { <operator>: <value> }
// filter: <field>: <value>
// filter: field: { $not: { <operator>: <value> } }
// filter: $and: [filter]  /   {{field1: value1}, {field2:{$op2, value2}}, {field3:{$op3, value3}}}
// filter: $or: [filter]
// filter: $nor: [filter]

type and struct {
	filters []Filter
}

func (a *and) ToBsonD() bson.D {
	// merge op; merge field; $and
	var values bson.A
	var fields bson.D
	fs := make(map[string]bool)
	var ops bson.D

	for _, f := range a.filters {
		ds := f.ToBsonD()
		for _, d := range ds {
			if d.Key == "$and" {
				if vs, ok := d.Value.(bson.A); ok {
					// todo re
					for _, vvs := range vs {
						if vvsds, ok := vvs.(bson.D); ok {
							for _, vvsd := range vvsds {
								if fs[vvsd.Key] {
									//
								}
							}
						} else {
							values = append(values, vvs)
						}
					}
				} else {
					values = append(values, ds)
				}
			} else if strings.HasPrefix(d.Key, "$") {
				values = append(values, ds)
			} else {

			}
		}
	}
}

type logic struct {
	operator string
	filters  []Filter
}

func (l *logic) ToBsonD() bson.D {

	var value bson.A

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

	return bson.D{{l.operator, value}}
}

//const (
//	or  = `$or`
//	and = `$and`
//)

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

func (l *nor) ToBsonD() bson.D {

	var value []bson.D

	for _, filter := range l.filters {
		value = append(value, filter.ToBsonD())
	}

	return bson.D{{`$nor`, value}}
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
