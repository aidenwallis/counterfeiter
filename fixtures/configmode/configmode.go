package configmode

import (
	"context"
	"io"
)

//go:generate go run github.com/aidenwallis/counterfeiter/v7

type Widget interface {
	Spin(speed int) error
}

type Gadget interface {
	Open(ctx context.Context, name string) (io.ReadCloser, error)
}
