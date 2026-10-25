package directives

//counterfeiter:generate . Thing
type Thing interface {
	Do(name string) (int, error)
}
