package the_aliased_package // import "github.com/aidenwallis/counterfeiter/v6/fixtures/aliased_package"

//go:generate go run github.com/aidenwallis/counterfeiter/v6 . InAliasedPackage
type InAliasedPackage interface {
	Stuff(int) string
}
