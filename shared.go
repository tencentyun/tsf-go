package tsf

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tencentyun/tsf-go/gin"
)

func ServerOperation(ctx context.Context) (method string, operation string) {
	method = "POST"
	if c, ok := gin.FromGinContext(ctx); ok {
		operation = c.Ctx.FullPath()
		method = c.Ctx.Request.Method
	} else if tr, ok := transport.FromServerContext(ctx); ok {
		operation = tr.Operation()
		if tr.Kind() == transport.KindHTTP {
			if ht, ok := tr.(*http.Transport); ok {
				operation = ht.PathTemplate()
				method = ht.Request().Method
			}
		}
	}
	return
}

func ClientOperation(ctx context.Context) (method string, operation string) {
	method = "POST"
	if tr, ok := transport.FromClientContext(ctx); ok {
		operation = tr.Operation()
		if tr.Kind() == transport.KindHTTP {
			if ht, ok := tr.(*http.Transport); ok {
				operation = ht.PathTemplate()
				method = ht.Request().Method
			}
		}
	}
	return
}
