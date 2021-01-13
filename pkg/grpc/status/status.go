package status

import (
	"github.com/tencentyun/tsf-go/pkg/statusError"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//FromGrpcStatus convert grpc to app error
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
		return statusError.BadRequest(gst.Message())
	case codes.NotFound:
		return statusError.NotFound(gst.Message())
	case codes.PermissionDenied:
		return statusError.Forbidden(gst.Message())
	case codes.Unauthenticated:
		return statusError.Unauthorized(gst.Message())
	case codes.ResourceExhausted:
		return statusError.LimitExceed(gst.Message())
	case codes.Unimplemented:
		return statusError.NotImplemented(gst.Message())
	case codes.Aborted:
		return statusError.ServerClosed(gst.Message())
	case codes.DeadlineExceeded:
		return statusError.Deadline(gst.Message())
	case codes.Unavailable:
		return statusError.ServiceUnavailable(gst.Message())
	case codes.FailedPrecondition:
		return statusError.PreconditionRequired(gst.Message())
	case codes.Unknown:
		return statusError.Unknown(gst.Message())
	default:
		return statusError.Internal(gst.Message())
	}
}

// ToGrpcStatus convert app error to grpc error
func ToGrpcStatus(err error) error {
	if ec, ok := err.(*statusError.StatusError); ok && ec != nil {
		if ec.Code() == statusError.CodeOK {
			return nil
		}
		return status.New(errCodeToGrpcCode(ec), ec.Reason()).Err()
	}

	return err
}

func errCodeToGrpcCode(ec *statusError.StatusError) codes.Code {
	// 系统错误
	switch ec.Code() {
	case statusError.CodeOK:
		return codes.OK
	case statusError.CodeBadRequest:
		return codes.InvalidArgument
	case statusError.CodeUnauthorized:
		return codes.Unauthenticated
	case statusError.CodeForbidden:
		return codes.PermissionDenied
	case statusError.CodeNotFound:
		return codes.NotFound
	case statusError.CodePreconditionRequired:
		return codes.FailedPrecondition
	case statusError.CodeLimitExceed:
		return codes.ResourceExhausted
	case statusError.CodeServerClosed, statusError.CodeClientClosed:
		return codes.Aborted
	case statusError.CodeInternal:
		return codes.Internal
	case statusError.CodeNotImplemented:
		return codes.Unimplemented
	case statusError.CodeServiceUnavailable:
		return codes.Unavailable
	case statusError.CodeDeadline:
		return codes.DeadlineExceeded
	case statusError.CodeUnknown:
		return codes.Unknown
	default:
		return codes.Internal
	}
}
