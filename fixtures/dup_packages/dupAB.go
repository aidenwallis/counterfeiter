package dup_packages // import "github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages"

//counterfeiter:generate . DupAB
type DupAB interface {
	DupA
	DupB
}
