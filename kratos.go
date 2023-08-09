package pjwt

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/ncuhome/PJWTC/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strconv"
	"strings"
)

type WithContext func(ctx context.Context, username int) context.Context

func KratosHandler(withContext WithContext) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				tokenString := tr.RequestHeader().Get("Authorization")
				if !strings.HasPrefix(tokenString, "passport ") {
					return nil, ErrTokenInvalid
				}
				tokenString = strings.TrimPrefix(tokenString, "passport ")
				var (
					result *proto.ParseJwtResult
					conn   *grpc.ClientConn
				)
				conn, err = grpc.Dial(Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					return nil, err
				}
				client := proto.NewPassportClient(conn)
				result, err = client.ParseJwt(context.TODO(), &proto.RequestParseJwt{
					Token: tokenString,
				})
				if err != nil {
					if errors.Is(err, grpc.ErrServerStopped) {
						return nil, err
					} else {
						return nil, err
					}
				}

				if !result.Valid || result.Claims == nil {
					return nil, ErrTokenInvalid
				}
				var userId int64
				userId, err = strconv.ParseInt(result.Claims.Xh, 10, 64)
				if err != nil {
					log.Info(err)
					return nil, ErrTokenInvalid
				}
				ctx = withContext(ctx, int(userId))
			}
			return handler(ctx, req)
		}
	}
}
