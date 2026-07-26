package pjwt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	proto "github.com/ncuhome/PJWT-Protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakePassportClient struct {
	parse func(context.Context, *proto.RequestParseJwt) (*proto.ParseJwtResult, error)
}

func (f fakePassportClient) ParseJwt(ctx context.Context, req *proto.RequestParseJwt, _ ...grpc.CallOption) (*proto.ParseJwtResult, error) {
	return f.parse(ctx, req)
}

func (fakePassportClient) GenToken(context.Context, *proto.RequestGenToken, ...grpc.CallOption) (*proto.GenTokenResult, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestHandlerUsesRequestDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var success bool
	middleware := &Middleware{
		handlers: Handlers{
			ParseError:  func(*gin.Context, error) { t.Fatal("ParseError called") },
			ServerError: func(*gin.Context, error) { t.Fatal("ServerError called") },
			Success: func(_ *gin.Context, uid uint64, xh string) {
				success = uid == 1001 && xh == "1001"
			},
		},
		client: fakePassportClient{parse: func(ctx context.Context, req *proto.RequestParseJwt) (*proto.ParseJwtResult, error) {
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("ParseJwt context has no deadline")
			}
			if req.Token != "token" {
				t.Fatalf("ParseJwt token = %q", req.Token)
			}
			return &proto.ParseJwtResult{Valid: true, Claims: &proto.Claims{Id: "1001", Xh: "1001"}}, nil
		}},
		requestTimeout: time.Second,
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request.Header.Set("Authorization", "Passport token")
	middleware.Handler()(ctx)
	if !success {
		t.Fatal("Success was not called with parsed claims")
	}
}

func TestHandlerRejectsMalformedAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parseErrors := 0
	middleware := &Middleware{
		handlers: Handlers{
			ParseError:  func(*gin.Context, error) { parseErrors++ },
			ServerError: func(*gin.Context, error) { t.Fatal("ServerError called") },
			Success:     func(*gin.Context, uint64, string) { t.Fatal("Success called") },
		},
		client: fakePassportClient{parse: func(context.Context, *proto.RequestParseJwt) (*proto.ParseJwtResult, error) {
			t.Fatal("ParseJwt called")
			return nil, nil
		}},
		requestTimeout: time.Second,
	}

	for _, authorization := range []string{"", "Bearer token", "passport ", "passport too many parts"} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		ctx.Request.Header.Set("Authorization", authorization)
		middleware.Handler()(ctx)
	}
	if parseErrors != 4 {
		t.Fatalf("ParseError calls = %d, want 4", parseErrors)
	}
}

func TestHandlerClassifiesAbortedAsParseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parseErrors := 0
	middleware := &Middleware{
		handlers: Handlers{
			ParseError:  func(*gin.Context, error) { parseErrors++ },
			ServerError: func(*gin.Context, error) { t.Fatal("ServerError called") },
			Success:     func(*gin.Context, uint64, string) { t.Fatal("Success called") },
		},
		client: fakePassportClient{parse: func(context.Context, *proto.RequestParseJwt) (*proto.ParseJwtResult, error) {
			return nil, status.Error(codes.Aborted, "invalid token")
		}},
		requestTimeout: time.Second,
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request.Header.Set("Authorization", "passport token")
	middleware.Handler()(ctx)
	if parseErrors != 1 {
		t.Fatalf("ParseError calls = %d, want 1", parseErrors)
	}
}

func TestNewWithConfigRequiresHandlers(t *testing.T) {
	if _, err := NewWithConfig(Handlers{}, Config{Addr: "localhost:80"}); err != ErrHandlersInvalid {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
}
