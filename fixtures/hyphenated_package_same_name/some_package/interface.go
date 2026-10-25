package some_package // import "github.com/aidenwallis/counterfeiter/v7/fixtures/hyphenated_package_same_name/some_package"

import "github.com/aidenwallis/counterfeiter/v7/fixtures/hyphenated_package_same_name/hyphen-ated/some_package"

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . SomeInterface
type SomeInterface interface {
	CreateThing() some_package.Thing
}
