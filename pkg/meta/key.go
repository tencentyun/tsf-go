package meta

import "strings"

const (
	PrefixDest   = "destination."
	PrefixSource = "source."
	PrefixUser   = "user_def."
)

const (
	ApplicationID      = "application.id"
	GroupID            = "group.id"
	ConnnectionIP      = "connection.ip"
	ApplicationVersion = "application.version"
	ServiceName        = "service.name"
	Interface          = "interface"
	RequestHTTPMethod  = "request.http.method"
	ServiceNamespace   = "service.namespace"
	Namespace          = "namespace"

	Tracer = "tsf.tracer"
	LaneID = "lane.id"
)

var carriedKey = map[string]struct{}{
	ApplicationID:      struct{}{},
	GroupID:            struct{}{},
	ConnnectionIP:      struct{}{},
	ApplicationVersion: struct{}{},
	ServiceName:        struct{}{},
	Interface:          struct{}{},
	RequestHTTPMethod:  struct{}{},
	ServiceNamespace:   struct{}{},
}

var linkKey = map[string]struct{}{
	LaneID: struct{}{},
}

func IsLinkKey(key string) (ok bool) {
	_, ok = linkKey[key]
	return
}

func IsIncomming(key string) (ok bool) {
	_, ok = carriedKey[key]
	return
}

func IsOutgoing(key string) (ok bool) {
	_, ok = carriedKey[key]
	if !ok {
		_, ok = linkKey[key]
	}
	return
}

func UserKey(key string) string {
	return PrefixUser + key
}

func GetUserKey(key string) string {
	return strings.TrimPrefix(key, PrefixUser)
}

func IsUserKey(key string) bool {
	return strings.HasPrefix(key, PrefixUser)
}

func SourceKey(key string) string {
	return PrefixSource + key
}

func DestKey(key string) string {
	return PrefixDest + key
}
