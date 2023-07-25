package pjwt

import "os"

var (
	Addr = os.Getenv("PJWT_ADDR")
)

func init() {
	if Addr == "" {
		Addr = "jwt-grpc.passport"
	}
}
