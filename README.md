# PJWTC

passport jwt 解析 Gin 中间件

默认使用集群内地址 `jwt-grpc.passport:80`，迁移期间保持明文 gRPC。服务端启用
TLS 后，客户端设置：

| 环境变量 | 说明 |
|---|---|
| `PJWT_ADDR` | 可选，PJWT 地址 |
| `PJWT_TLS` | 设为 `true` 启用 TLS 1.3 |
| `PJWT_CA_FILE` | 可选，私有 CA PEM 文件 |
| `PJWT_SERVER_NAME` | 可选，覆盖证书中的服务端名称 |
| `PJWT_TIMEOUT` | 可选，单次验证超时，默认 `2s` |

### 使用中间件

```go
package main

import (
    "github.com/gin-gonic/gin"
    pjwt "github.com/ncuhome/PJWTC"
    "log"
)

func main() {
	middleware, err := pjwt.New(pjwt.Handlers{
		ParseError: func(c *gin.Context, err error) {
			c.AbortWithStatus(401)
		},
		ServerError: func(c *gin.Context, err error) {
			c.AbortWithStatus(500)
		},
		Success: func(c *gin.Context, uid uint64, xh string) {
			c.Set("uid", uid)
			c.Set("xh", xh)
		},
	})
	if err != nil {
		log.Fatalln("初始化鉴权中间件失败:", err)
	}
	defer middleware.Close()

	router := gin.Default()
	router.Use(middleware.Handler())
	_ = router.Run()
}

```
