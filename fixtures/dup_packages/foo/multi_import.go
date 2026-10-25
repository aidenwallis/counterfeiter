package foo // import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/foo"

import (
	"github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/a/foo"
	bfoo "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/b/foo"
)

type S struct{}

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . MultiAB
type MultiAB interface {
	Mine() S
	foo.I
	bfoo.I
}
