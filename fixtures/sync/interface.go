package sync // import "github.com/aidenwallis/counterfeiter/v7/fixtures/sync"

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . SyncSomething
type SyncSomething interface {
	DoThings(string, uint64) (int, error)
	DoNothing()
	DoASlice([]byte)
	DoAnArray([4]byte)
}
