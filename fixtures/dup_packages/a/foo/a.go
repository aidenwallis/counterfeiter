package foo // import "github.com/aidenwallis/counterfeiter/v6/fixtures/dup_packages/a/foo"

type S struct{}

//go:generate go run github.com/aidenwallis/counterfeiter/v6 . I
type I interface {
	FromA() S
}
