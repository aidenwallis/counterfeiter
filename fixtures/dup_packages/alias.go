package dup_packages // import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages"

import (
	"github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/a"
	afoo "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/a/foo"
	"github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/b/foo"
)

//go:generate go run github.com/aidenwallis/counterfeiter/v7 -generate
//counterfeiter:generate . AliasV1
type AliasV1 interface {
	a.A
	afoo.I
	foo.I
}
