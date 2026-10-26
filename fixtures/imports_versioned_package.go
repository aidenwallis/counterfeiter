package fixtures

import (
	"github.com/aidenwallis/counterfeiter/v7/fixtures/versioned/v1"
)

//counterfeiter:generate . ImportsVersionedPackage
type ImportsVersionedPackage interface {
	UseVersionedType(v1.Thing)
}
