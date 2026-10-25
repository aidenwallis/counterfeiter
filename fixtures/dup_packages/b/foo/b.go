package foo // import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/b/foo"

type S struct{}

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . I
type I interface {
	FromB() S
}
