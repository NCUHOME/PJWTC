# PJWTC

passport jwt 解析中间件

此仓库为 Gin 中间件版本，另有 Kratos 中间件：[PJWTC-Kratos](https://github.com/ncuhome/PJWTC-Kratos)

proto 更新重新生成命令:

```shell
protoc --go_out=. --go-grpc_out=. ./proto/api.proto
```

### 使用中间件

`$env:GOPRIVATE="github.com/ncuhome"`

默认值使用集群内连接地址，如需覆盖，可以设置环境变量 `PJWT_ADDR`

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