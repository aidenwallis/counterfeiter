package another_package // import "github.com/aidenwallis/counterfeiter/v7/fixtures/another_package"

type SomeType int

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . AnotherInterface
type AnotherInterface interface {
	AnotherMethod([]SomeType, map[SomeType]SomeType, *SomeType, SomeType, chan SomeType)
}
