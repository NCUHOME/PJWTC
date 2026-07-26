package pjwt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/gin-gonic/gin"
	proto "github.com/ncuhome/PJWT-Protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	maxGRPCMessageSize = 32 * 1024
	maxTokenLength     = 16 * 1024
)

// Handlers 需要实现全部字段
type Handlers struct {
	// err 可能为 nil
	ParseError  func(c *gin.Context, err error)
	ServerError func(c *gin.Context, err error)
	Success     func(c *gin.Context, uid uint64, xh string)
}

func New(handlers Handlers) (*Middleware, error) {
	config, err := configFromEnv()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(handlers, config)
}

func NewWithConfig(handlers Handlers, config Config) (*Middleware, error) {
	if handlers.ParseError == nil || handlers.ServerError == nil || handlers.Success == nil {
		return nil, ErrHandlersInvalid
	}
	if config.Addr == "" {
		return nil, errors.New("PJWT address is required")
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = defaultRequestTimeout
	}

	transportCredentials, err := clientCredentials(config)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(
		config.Addr,
		grpc.WithTransportCredentials(transportCredentials),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxGRPCMessageSize),
			grpc.MaxCallSendMsgSize(maxGRPCMessageSize),
		),
	)
	if err != nil {
		return nil, err
	}

	return &Middleware{
		handlers:       handlers,
		client:         proto.NewPassportClient(conn),
		conn:           conn,
		requestTimeout: config.RequestTimeout,
	}, nil
}

func clientCredentials(config Config) (credentials.TransportCredentials, error) {
	if !config.TLS {
		return insecure.NewCredentials(), nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ServerName: config.ServerName,
	}
	if config.CAFile != "" {
		pem, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, err
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		if roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("PJWT CA file contains no certificates")
		}
		tlsConfig.RootCAs = roots
	}
	return credentials.NewTLS(tlsConfig), nil
}

type Middleware struct {
	handlers       Handlers
	client         proto.PassportClient
	conn           *grpc.ClientConn
	requestTimeout time.Duration
}

func (a *Middleware) Close() error {
	if a.conn == nil {
		return nil
	}
	return a.conn.Close()
}

func (a *Middleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		scheme, token, found := strings.Cut(authorization, " ")
		if !found || !strings.EqualFold(scheme, "passport") || token == "" || len(token) > maxTokenLength || strings.Contains(token, " ") {
			a.handlers.ParseError(c, ErrTokenInvalid)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), a.requestTimeout)
		defer cancel()
		result, err := a.client.ParseJwt(ctx, &proto.RequestParseJwt{
			Token: token,
		})
		if err != nil {
			if errors.Is(err, grpc.ErrServerStopped) || status.Code(err) != codes.Aborted {
				a.handlers.ServerError(c, err)
			} else {
				a.handlers.ParseError(c, err)
			}
			return
		}

		if result == nil || !result.Valid || result.Claims == nil {
			a.handlers.ParseError(c, ErrTokenInvalid)
			return
		}

		uid, err := strconv.ParseUint(result.Claims.Id, 10, 64)
		if err != nil {
			a.handlers.ServerError(c, err)
			return
		}

		a.handlers.Success(c, uid, result.Claims.Xh)
	}
}
