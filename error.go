package pjwt

import "errors"

var ErrTokenInvalid = errors.New("token format invalid")

var ErrHandlersInvalid = errors.New("all middleware handlers are required")
