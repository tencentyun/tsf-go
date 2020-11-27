package meta

import "context"

type sysKey struct{}
type userKey struct{}

type SysMeta map[string]interface{}
type UserMeta map[string]string

type SysPair struct {
	Key   string
	Value interface{}
}

type UserPair struct {
	Key   string
	Value string
}

func WithUser(ctx context.Context, pairs ...UserPair) context.Context {
	origin, _ := ctx.Value(userKey{}).(UserMeta)
	copied := make(UserMeta, len(origin)+len(pairs))
	for k, v := range origin {
		copied[k] = v
	}
	for _, pair := range pairs {
		copied[pair.Key] = pair.Value
	}
	return context.WithValue(ctx, userKey{}, copied)
}

func WithSys(ctx context.Context, pairs ...SysPair) context.Context {
	origin, _ := ctx.Value(sysKey{}).(SysMeta)
	copied := make(SysMeta, len(origin)+len(pairs))
	for k, v := range origin {
		copied[k] = v
	}
	for _, pair := range pairs {
		copied[pair.Key] = pair.Value
	}
	return context.WithValue(ctx, sysKey{}, copied)
}

type Context struct {
	context.Context
}

func RangeSys(ctx context.Context, f func(key string, value interface{})) {
	origin, ok := ctx.Value(sysKey{}).(SysMeta)
	if !ok || origin == nil {
		return
	}
	for k, v := range origin {
		f(k, v)
	}
}

func RangeUser(ctx context.Context, f func(key string, value string)) {
	origin, ok := ctx.Value(userKey{}).(UserMeta)
	if !ok || origin == nil {
		return
	}
	for k, v := range origin {
		f(k, v)
	}
}

// TODO: remove to internal package
func Sys(ctx context.Context, key string) (res interface{}) {
	origin, ok := ctx.Value(sysKey{}).(SysMeta)
	if !ok || origin == nil {
		return
	}
	res = origin[key]
	return
}

func User(ctx context.Context, key string) (res string) {
	origin, ok := ctx.Value(userKey{}).(UserMeta)
	if !ok || origin == nil {
		return
	}
	res = origin[key]
	return
}
