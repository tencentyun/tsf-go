package tag

import (
	"context"
)

type Relation int32

const (
	AND       Relation = 0
	OR        Relation = 1
	COMPOSITE Relation = 2
)

type Rule struct {
	ID         string
	Name       string
	Tags       []Tag
	Expression Relation
}

func (r *Rule) Hit(ctx context.Context) bool {
	if len(r.Tags) == 0 {
		return true
	} else if r.Expression == AND {
		for _, tag := range r.Tags {
			if !tag.Hit(ctx) {
				return false
			}
		}
		return true
	} else if r.Expression == OR {
		for _, tag := range r.Tags {
			if tag.Hit(ctx) {
				return true
			}
		}
	} else if r.Expression == COMPOSITE {
		// TODO: impl
		return false
	}
	return false
}
