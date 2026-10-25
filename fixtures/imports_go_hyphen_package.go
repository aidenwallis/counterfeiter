package fixtures

import (
	"github.com/aidenwallis/counterfeiter/v7/fixtures/go-hyphenpackage"
)

//counterfeiter:generate . ImportsGoHyphenPackage
type ImportsGoHyphenPackage interface {
	UseHyphenType(hyphenpackage.HyphenType)
}
