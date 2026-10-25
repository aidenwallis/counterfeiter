package nested

//counterfeiter:generate -o fake . Nest
type Nest interface {
	Hatch(eggs ...string)
}
