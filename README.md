# PJWTC

passport jwt 解析中间件

proto 更新重新生成命令:

```shell
protoc --go_out=. --go-grpc_out=. ./proto/api.proto
```

### 使用中间件

`$env:GOPRIVATE="github.com/ncuhome"`

默认值使用集群内连接地址，如需覆盖，可以设置环境变量 `PJWT_ADDR`

## 在 Gin 中使用
```go
package middlewares

import (
    "github.com/gin-gonic/gin"
    pjwt "github.com/ncuhome/PJWTC"
	"log"
)

func Auth() gin.HandlerFunc {
	middleware, e := pjwt.New(pjwt.Handlers{
		ParseError: func(c *gin.Context, err error) {
			c.AbortWithStatus(401)
		},
		ServerError: func(c *gin.Context, err error) {
			c.AbortWithStatus(500)
		},
		Success: func(c *gin.Context, xh string) {
			c.Set("xh", xh)
		},
	})
	if e != nil {
		log.Fatalln("初始化鉴权中间件失败:", e)
	}
	
	return middleware.Handler()
}

```
## 在 Kratos 中使用

```go
package server

import (
	"context"
	pjwt "github.com/ncuhome/PJWTC"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	v1 "example/api/helloworld/v1"
	"example/internal/conf"
	"example/internal/service"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, s *service.ExampleService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			selector.Server(pjwt.KratosHandler(func(ctx context.Context, username int) context.Context {
				return context.WithValue(ctx, "UserID", username)
			})).Match(NewSkipRoutersMatcher()).Build(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterExampleHTTPServer(srv, s)
	return srv
}

func NewSkipRoutersMatcher() selector.MatchFunc {
	skipRouters := map[string]struct{}{}

	return func(ctx context.Context, operation string) bool {
		if _, ok := skipRouters[operation]; ok {
			return false
		}
		return true
	}
}
```