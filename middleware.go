package pjwt

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/ncuhome/PJWTC/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
)

// Handlers 需要实现全部字段
type Handlers struct {
	// err 可能为 nil
	ParseError  func(c *gin.Context, err error)
	ServerError func(c *gin.Context, err error)
	Success     func(c *gin.Context, xh string)
}

func New(handlers Handlers) (*Middleware, error) {
	conn, err := grpc.Dial(Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Middleware{
		handlers: handlers,
		client:   proto.NewPassportClient(conn),
	}, nil
}

type Middleware struct {
	handlers Handlers
	client   proto.PassportClient
}

func (a *Middleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if !strings.HasPrefix(token, "passport ") {
			a.handlers.ParseError(c, ErrTokenInvalid)
			return
		}
		token = strings.TrimPrefix(token, "passport ")

		result, err := a.client.ParseJwt(context.TODO(), &proto.RequestParseJwt{
			Token: token,
		})
		if err != nil {
			if err == grpc.ErrServerStopped {
				a.handlers.ServerError(c, err)
			} else {
				a.handlers.ParseError(c, err)
			}
			return
		}

		if !result.Valid || result.Claims == nil {
			a.handlers.ParseError(c, ErrTokenInvalid)
			return
		}

		a.handlers.Success(c, result.Claims.Xh)
	}
}
