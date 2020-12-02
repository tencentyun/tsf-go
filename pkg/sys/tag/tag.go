package tag

import (
	"context"
	"regexp"
	"strings"

	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/meta"
	"go.uber.org/zap"
)

type TagType int32

const (
	TypeSys  TagType = 0
	TypeUser TagType = 1

	Equal    = "EQUAL"
	NotEqual = "NOT_EQUAL"
	In       = "IN"
	NotIn    = "NOT_IN"
	Regex    = "REGEX"
)

// Tag is tsf tag
type Tag struct {
	Type     TagType
	Field    string
	Operator string
	Value    string
}

func (t Tag) Hit(ctx context.Context) bool {
	var v interface{}
	if t.Type == TypeSys {
		v = meta.Sys(ctx, t.Field)
		log.Debug(ctx, "hit sys:", zap.String("field", t.Field), zap.Any("value", v))
	} else {
		v = meta.User(ctx, t.Field)
		log.Debug(ctx, "hit user:", zap.String("field", t.Field), zap.Any("value", v))
	}
	if v == nil {
		return false
	}
	target, ok := v.(string)
	if !ok {
		return false
	}
	if t.Operator == Equal {
		return target == t.Value
	} else if t.Operator == NotEqual {
		return !(target == t.Value)
	} else if t.Operator == In {
		return strings.Contains(t.Value, target)
	} else if t.Operator == NotIn {
		return !strings.Contains(t.Value, target)
	} else if t.Operator == Regex {
		ok, _ = regexp.MatchString(t.Value, target)
		return ok
	}
	return false
}
