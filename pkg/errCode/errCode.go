package errCode

import (
	"encoding/json"
	"fmt"

	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	// 根据https://httpstatuses.com/定义
	// 小于1000的是系统内置的报错信息，业务代码也可以使用，但是Equal()比较时候只会比较code码，不会严格比较错误信息
	OK                   = New(200, "200 OK")
	BadRequest           = New(400, "400 Bad Request")
	Unauthorized         = New(401, "401 Unauthorized")
	Forbidden            = New(403, "403 Forbidden")
	NotFound             = New(404, "404 Not Found")
	PreconditionRequired = New(428, "428 Precondition Required")
	LimitExceed          = New(429, "429 Too Many Requests")
	ServerClosed         = New(444, "444 Server Closed")
	ClientClosed         = New(499, "499 Client Closed")
	Internal             = New(500, "500 Internal Server Error")
	NotImplemented       = New(501, "501 Not Implemented")
	ServiceUnavailable   = New(503, "503 Service Unavailable")
	Deadline             = New(504, "504 Timeout")

	// CustomErr 1000为系统报错和业务报错分界点
	// 大于或等于1000的是业务自定义错误，当使用Equal()比较时会严格比较具体报错信息
	CustomErr = New(1000, "1000 CustomErr")
)

type ErrCode struct {
	s *spb.Status
}

func New(code int, format string, a ...interface{}) ErrCode {
	return ErrCode{s: &spb.Status{Code: int32(code), Message: fmt.Sprintf(format, a...)}}
}

func (ec ErrCode) MarshalJSON() ([]byte, error) {
	if ec.s == nil {
		return nil, nil
	}
	temp := ec.s.Details
	ec.s.Details = nil
	content, err := json.Marshal(ec.s)
	ec.s.Details = temp
	return content, err
}

func (ec *ErrCode) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, ec)
}

func (ec ErrCode) Error() string {
	if ec.s == nil {
		return ""
	}
	return ec.s.Message
}

func (ec ErrCode) Code() int {
	if ec.s == nil {
		return 200
	}
	return int(ec.s.Code)
}

func (ec ErrCode) Equal(err error) bool {
	if err == nil {
		if ec.Code() == 200 {
			return true
		}
		return false
	}
	if ec2, ok := err.(ErrCode); ok && ec.Code() == ec2.Code() {
		if ec.Code() < 1000 {
			// 系统内置的错误，不用比较报错信息
			return true
		}
		return ec.Error() == ec2.Error()
	}
	return false
}

func (ec *ErrCode) WithDetails(details ...*anypb.Any) {
	if ec.s == nil {
		return
	}
	ec.s.Details = append(ec.s.Details, details...)
}

func (ec ErrCode) Details() []*anypb.Any {
	if ec.s == nil {
		return nil
	}
	return ec.s.GetDetails()
}
