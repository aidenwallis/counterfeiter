package the_aliased_package // import "github.com/aidenwallis/counterfeiter/v7/fixtures/aliased_package"

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . InAliasedPackage
type InAliasedPackage interface {
	Stuff(int) string
}
