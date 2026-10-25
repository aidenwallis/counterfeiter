package type_aliases // import "github.com/aidenwallis/counterfeiter/v6/fixtures/type_aliases"

import (
	"context"

	"github.com/aidenwallis/counterfeiter/v6/fixtures/type_aliases/extra"
)

//go:generate go run github.com/aidenwallis/counterfeiter/v6 -generate

//counterfeiter:generate . WithAliasedType
type WithAliasedType interface {
	FindExample(ctx context.Context, filter extra.M) ([]string, error)
}
