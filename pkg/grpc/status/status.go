package status

import (
	"github.com/tencentyun/tsf-go/pkg/errCode"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func FromGrpcStatus(err error) error {
	gst, ok := status.FromError(err)
	if !ok || gst == nil {
		return err
	}
	gcode := gst.Code()
	switch gcode {
	case codes.OK:
		return nil
	case codes.InvalidArgument:
		return errCode.New(errCode.BadRequest.Code(), gst.Err().Error())
	case codes.NotFound:
		return errCode.New(errCode.NotFound.Code(), gst.Err().Error())
	case codes.PermissionDenied:
		return errCode.New(errCode.Forbidden.Code(), gst.Err().Error())
	case codes.Unauthenticated:
		return errCode.New(errCode.Unauthorized.Code(), gst.Err().Error())
	case codes.ResourceExhausted:
		return errCode.New(errCode.LimitExceed.Code(), gst.Err().Error())
	case codes.Unimplemented:
		return errCode.New(errCode.NotImplemented.Code(), gst.Err().Error())
	case codes.Aborted:
		return errCode.New(errCode.ServerClosed.Code(), gst.Err().Error())
	case codes.DeadlineExceeded:
		return errCode.New(errCode.Deadline.Code(), gst.Err().Error())
	case codes.Unavailable:
		return errCode.New(errCode.ServiceUnavailable.Code(), gst.Err().Error())
	case codes.FailedPrecondition:
		return errCode.New(errCode.PreconditionRequired.Code(), gst.Err().Error())
	case codes.Unknown:
		var ec *errCode.ErrCode
		e := ec.UnmarshalJSON([]byte(gst.Message()))
		if e == nil {
			return ec
		}
		return err
	default:
		return errCode.New(errCode.Internal.Code(), gst.Err().Error())
	}
}

func ToGrpcStatus(err error) error {
	if ec, ok := err.(errCode.ErrCode); ok {
		if ec.Code() == errCode.OK.Code() {
			return nil
		}
		var st *status.Status
		if ec.Code() <= errCode.CustomErr.Code() {
			st = status.New(errCodeToGrpcCode(ec), ec.Error())
		} else {
			eCon, _ := ec.MarshalJSON()
			st = status.New(codes.Unknown, string(eCon))
		}
		if ec.Details() != nil {
			// TODO: add more details
		}
		return st.Err()
	}
	return err
}

func errCodeToGrpcCode(ec errCode.ErrCode) codes.Code {
	// 系统错误
	switch ec.Code() {
	case errCode.OK.Code():
		return codes.OK
	case errCode.BadRequest.Code():
		return codes.InvalidArgument
	case errCode.Unauthorized.Code():
		return codes.Unauthenticated
	case errCode.Forbidden.Code():
		return codes.PermissionDenied
	case errCode.NotFound.Code():
		return codes.NotFound
	case errCode.PreconditionRequired.Code():
		return codes.FailedPrecondition
	case errCode.LimitExceed.Code():
		return codes.ResourceExhausted
	case errCode.ServerClosed.Code(), errCode.ClientClosed.Code():
		return codes.Aborted
	case errCode.Internal.Code():
		return codes.Internal
	case errCode.NotImplemented.Code():
		return codes.Unimplemented
	case errCode.ServiceUnavailable.Code():
		return codes.Unavailable
	case errCode.Deadline.Code():
		return codes.DeadlineExceeded
	}
	return codes.Internal
}
