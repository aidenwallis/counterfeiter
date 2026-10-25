package fixtures

import (
	"github.com/aidenwallis/counterfeiter/v7/fixtures/go-hyphenpackage"
)

//counterfeiter:generate . GenericImportedConstraint
type GenericImportedConstraint[T hyphenpackage.Hyphenated] interface {
	ReturnT() T
	DoSomething()
}
