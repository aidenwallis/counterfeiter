package a // import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/a"

import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/a/foo"

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . A
type A interface {
	V1() foo.I
}
