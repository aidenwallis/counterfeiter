package internalpkg

import "github.com/aidenwallis/counterfeiter/v6/fixtures/internalpkg/internal"

//go:generate go run github.com/aidenwallis/counterfeiter/v6 -generate
//counterfeiter:generate . Context

type Context = internal.Context
