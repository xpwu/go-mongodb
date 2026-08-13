package projection

import (
	"github.com/xpwu/go-mongodb/field"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ExcludeBuilder struct {
	proj bson.D
	seen map[string]bool
}

func Exclude(fs ...field.Field) *ExcludeBuilder {
	b := &ExcludeBuilder{seen: make(map[string]bool)}
	for _, f := range fs {
		if ok := b.seen[f.FullName()]; !ok {
			b.seen[f.FullName()] = true
			b.proj = append(b.proj, bson.E{Key: f.FullName(), Value: 0})
		}
	}
	return b
}

func (b *ExcludeBuilder) Build() bson.D {
	return b.proj
}
