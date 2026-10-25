package dup_packages // import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages"

import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/a/foo"

//counterfeiter:generate . DupA
type DupA interface {
	A() foo.S
}
