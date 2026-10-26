# `counterfeiter` [![GitHub Actions](https://github.com/aidenwallis/counterfeiter/actions/workflows/go.yml/badge.svg)](https://github.com/aidenwallis/counterfeiter/actions/workflows/go.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/aidenwallis/counterfeiter/v7.svg)](https://pkg.go.dev/github.com/aidenwallis/counterfeiter/v7)

Go code declares what it needs from its dependencies as interfaces, usually small ones
defined by the package that uses them. Testing that code means supplying fake
implementations of those interfaces. Go has no way to build one at runtime, so they are
written by hand or generated.

`counterfeiter` generates test doubles for a given interface. Given this:

```go
package foo

//go:generate go tool counterfeiter -generate

//counterfeiter:generate . MySpecialInterface
type MySpecialInterface interface {
	DoThings(string, uint64) (int, error)
}
```

`go generate` writes `foofakes/fake_my_special_interface.go`, and your tests can do this:

```go
fake := &foofakes.FakeMySpecialInterface{}
fake.DoThingsReturns(3, nil)

num, err := fake.DoThings("stuff", 5)

Expect(num).To(Equal(3))
Expect(fake.DoThingsCallCount()).To(Equal(1))
str, _ := fake.DoThingsArgsForCall(0)
Expect(str).To(Equal("stuff"))
```

## Getting Started

`counterfeiter` is run by [`go generate`](https://go.dev/blog/generate), so fakes are regenerated alongside the code they fake. The steps below assume go 1.24 or later, which added `go tool`; for older versions of go, refer to an [older version of this README](https://github.com/maxbrunsfeld/counterfeiter/blob/e39cbe6aaa94a0b6718cf3d413cd5319c3a1f6fa/README.md#using-counterfeiter).

### Step 1 - Add `counterfeiter` as a tool dependency

Establish a tool dependency on counterfeiter by running the following command:

```shell
go get -tool github.com/aidenwallis/counterfeiter/v7
```

### Step 2 - Add directives

You need two kinds of directive: **one** `go:generate` line for the whole module, which runs `counterfeiter -generate`, and a `counterfeiter:generate` line for each interface you want a fake for.

Put the `go:generate` line in the package at the root of your module, in a file of its own if you like:

```go
// generate.go, at the root of the module
package myapp

//go:generate go tool counterfeiter -generate
```

That is the only `go:generate` line you add. When it runs, `counterfeiter` reads every package in the module, the root and every package below it, for `counterfeiter:generate` directives, and generates all of their fakes in one process.

Then put a `counterfeiter:generate` directive next to each interface you want faked, in whichever package declares it:

```go
// internal/store/store.go
package store

//counterfeiter:generate . Repository
type Repository interface {
	Get(id string) (Item, error)
}

//counterfeiter:generate . Cache
type Cache interface {
	Fetch(key string) ([]byte, bool)
}
```

```go
// api/api.go
package api

//counterfeiter:generate . Presenter
type Presenter interface {
	Show(Item) error
}
```

A `counterfeiter:generate` directive takes the same arguments as the command line, relative to the directory of the file it is in: the directory of the package that declares the interface (`.` for the package the directive is in) and the name of the interface, plus any of the flags described below.

A few things are left out of the search:

- A package with a `-generate` line of its own is left to that line, along with the packages below it, so a module that already has one such line per package keeps working.
- Nested modules (directories with their own `go.mod`) are skipped; give each one its own `go:generate` line.
- As with `go` itself, `testdata` and `vendor` directories and directories starting with `.` or `_` are skipped.

### Step 3 - Run `go generate`

Run `go generate` from the root of your module:

```shell
$ go generate ./...
Writing `FakeRepository` to `internal/store/storefakes/fake_repository.go`... Done
Writing `FakeCache` to `internal/store/storefakes/fake_cache.go`... Done
Writing `FakePresenter` to `api/apifakes/fake_presenter.go`... Done
```

Every fake goes next to the package that declares its interface. Run `go generate ./...` again whenever an interface changes.

In a large module, turning off the garbage collector makes generating hundreds of fakes noticeably faster, in exchange for more memory while it runs:

```shell
$ GOGC=off go generate ./...
```

## Defaults

`MySpecialInterface` above is declared in package `foo`, so `//counterfeiter:generate . MySpecialInterface` produces:

| | default | change it with |
|---|---|---|
| fake type | `FakeMySpecialInterface` (`Fake` + the interface name) | `-fake-name`, `-fake-name-template` |
| file | `fake_my_special_interface.go` (the fake's name in snake case) | `-o <path>.go` |
| directory | `foofakes/` (`<package>fakes`, beside the package that declares the interface) | `-o <dir>` |
| package | `foofakes` (`<package>fakes`), or whatever package already lives in the output directory | `-o`, `-test`, or a file declaring the package |

Every fake ends with a compile-time assertion that it satisfies the interface, so when the interface changes, the stale fake stops compiling until you run `go generate` again.

## Common Setups

The examples below are `counterfeiter:generate` directives, and assume the module has the one `//go:generate go tool counterfeiter -generate` line from Step 2.

### Shared settings for every directive

Flags given alongside `-generate` on the `//go:generate` line are the defaults for every `//counterfeiter:generate` directive it runs: `-o`, `-header`, `-q`, `-test` and `-fake-name-template`. A directive's own flags take precedence. `-o` is relative to each directive's directory, and `-header` to the `go:generate` line. So if you would rather keep each package's fakes in a `fake` package inside it, named after their interfaces, you can write that once:

```go
// generate.go, at the root of the module
//go:generate go tool counterfeiter -generate -o fake -fake-name-template '{{.TargetName}}'
```

```go
// internal/store/store.go
//counterfeiter:generate . Repository
//counterfeiter:generate . Cache
```

```shell
$ go generate ./...
Writing `Repository` to `internal/store/fake/repository.go`... Done
Writing `Cache` to `internal/store/fake/cache.go`... Done
```

### Interfaces from other packages, the standard library or third-party modules

You can fake any interface your module can import. A third-party module has to be a dependency first (`go get` it, so it appears in your `go.mod`), since `counterfeiter` loads the interface from the module cache the same way the compiler would. Name the interface as `<package-path>.<interface>`, or give the directory of the package and the interface name:

```go
//counterfeiter:generate io.WriteCloser
//counterfeiter:generate github.com/redis/go-redis/v9.Pipeliner
//counterfeiter:generate ../otherpackage OtherInterface
```

The two forms differ in where the fake goes. With `<package-path>.<interface>`, it goes into the fakes package of the directory the directive is in, so in package `foo` the first line writes `foofakes/fake_write_closer.go`. With a directory and an interface name, it goes into the fakes package beside that directory, so the third line writes `../otherpackage/otherpackagefakes/fake_other_interface.go`. Use `-o` to put it somewhere else.

### Fakes for tests inside the interface's own package

By default the fake lives in a sibling `<package>fakes` package. Tests in `<package>` itself cannot import it, because `<package>fakes` imports `<package>` and that would be an import cycle. To use a fake in those tests, point `-o` at the interface's own directory:

```go
//counterfeiter:generate -o . . MySpecialInterface
```

When the output directory is the directory of the package that declares the interface, `counterfeiter` generates the fake as a member of that package: it does not import the package, refers to its types unqualified, and can fake unexported interfaces too, in which case the fake is unexported as well (`gadget` gets `fakeGadget`) unless `-fake-name` names it. `-o` may also name a file in that directory, for example `-o fake_my_special_interface_test.go` to keep the fake out of the non-test build.

### Fakes for tests in `<package>_test`

Go lets a directory hold a second package for tests, `<package>_test`, which imports `<package>` and sees only its exported API. If your tests are in it, `-test` generates the fake into that external test package instead, as a `_test.go` file in the current directory, next to the tests that use it:

```go
//counterfeiter:generate -test . MySpecialInterface
//counterfeiter:generate -test ../otherpackage OtherInterface
//counterfeiter:generate -test io.WriteCloser
```

In package `foo`, all three of these write a `_test.go` file into the current directory, in package `foo_test`: `fake_my_special_interface_test.go`, `fake_other_interface_test.go` and `fake_write_closer_test.go`. With `-test` the fake goes into the package the directive is in, not the package that declares the interface, so faking `OtherInterface` writes nothing into `../otherpackage`. The fakes are only compiled for tests, and the tests use them unqualified (`&FakeMySpecialInterface{}`). The interface's package is imported as usual, so the interface must be exported. With `-o <dir>` the fake goes into the external test package of that directory instead.

### A fakes package whose name is not its directory name

If the output directory already contains Go files, the fake joins that package. If it is empty, the fake's package is named after the directory. So to name the package differently from its directory, say `impl_fakes` in `fakes/`, add a file declaring that package first:

```go
// fakes/doc.go
package impl_fakes
```

```go
//counterfeiter:generate -o fakes . MyInterface
```

### Naming fakes

`-fake-name` names one fake. `-fake-name-template` is a Go `text/template` in which `{{.TargetName}}` is the name of the interface being faked (first letter upper-cased); it applies wherever `-fake-name` is not given, and can be set once for the package on the `-generate` line. The file name follows the fake's name.

```go
//counterfeiter:generate -fake-name Repo . MyRepository
//counterfeiter:generate -fake-name-template '{{.TargetName}}Double' . MyPresenter
```

```shell
$ go generate ./...
Writing `Repo` to `foofakes/repo.go`... Done
Writing `MyPresenterDouble` to `foofakes/my_presenter_double.go`... Done
```

### A header on every fake

`-header` prepends the contents of a file to every generated fake, for a licence header for example. Set it once on the `-generate` line, relative to that line, or per directive.

```go
//go:generate go tool counterfeiter -generate -header hack/LICENSE.header
```

### Faking a function type

Function types can be faked as well. The fake is a struct whose `Spy` method has the function's signature, so `fake.Spy` goes wherever the function is expected. `Returns`, `CallCount`, `ArgsForCall` and the rest work as for a method, without a method name in front.

```go
//counterfeiter:generate . RequestHandler
type RequestHandler func(*http.Request) error
```

```go
fake := &foofakes.FakeRequestHandler{}
fake.Returns(nil)

server := NewServer(fake.Spy)
```

### Faking a whole package

Package mode, `-p`, is for code that calls package-level functions directly, `os.Hostname()` say, and has no interface to fake. It writes a file with an interface whose methods are the package's exported functions, and a shim struct that forwards each method to the package. The file carries a `counterfeiter:generate` directive of its own, so the next `go generate` produces a fake of that interface.

```go
//counterfeiter:generate -p os
```

```shell
$ go generate ./...
Writing `Os` to `osshim/os.go`... Done
$ go generate ./...
Writing `FakeOs` to `osshimfakes/fake_os.go`... Done
```

Your code takes the `osshim.Os` interface, production code passes `&osshim.OsShim{}`, tests pass the fake.

### Printing to stdout, quieter output

A trailing `-` prints the fake to standard output instead of writing a file:

```shell
$ go tool counterfeiter . MySpecialInterface -
```

`-q` drops the `Writing ...` lines, which is useful in a `-generate` line for a package with many fakes.

## Using Test Doubles In Your Tests

Instantiate fakes:

```go
import "my-repo/path/to/foo/foofakes"

var fake = &foofakes.FakeMySpecialInterface{}
```

For each method `<Method>` of the interface, the fake has:

| | |
|---|---|
| `<Method>Returns`, `<Method>ReturnsOnCall` | stub what it returns |
| `<Method>Calls`, `<Method>Stub` | replace it with a function |
| `<Method>CallCount`, `<Method>ArgsForCall` | inspect the calls it received |

You can stub return values:

```go
fake.DoThingsReturns(3, errors.New("the-error"))

num, err := fake.DoThings("stuff", 5)
Expect(num).To(Equal(3))
Expect(err).To(Equal(errors.New("the-error")))
```

or stub them for one call at a time, counting from zero; calls without an entry fall back to `Returns`:

```go
fake.DoThingsReturnsOnCall(0, 1, nil)
fake.DoThingsReturnsOnCall(1, 0, errors.New("the-error"))
```

When the result depends on the arguments, give the fake a function instead. It takes precedence over `Returns` and `ReturnsOnCall`, and calling either of those clears it again:

```go
fake.DoThingsCalls(func(s string, n uint64) (int, error) {
	if s == "stuff" {
		return 3, nil
	}
	return 0, errors.New("unexpected")
})
```

`DoThingsCalls` stores the function in the exported `DoThingsStub` field while holding the fake's lock, so it is safe even if other goroutines are already calling the fake. Setting the field directly is only useful in a struct literal, `&foofakes.FakeMySpecialInterface{DoThingsStub: ...}`. Without a stub or stubbed return values, the fake returns zero values.

Fakes record the arguments they were called with:

```go
fake.DoThings("stuff", 5)

Expect(fake.DoThingsCallCount()).To(Equal(1))

str, num := fake.DoThingsArgsForCall(0)
Expect(str).To(Equal("stuff"))
Expect(num).To(Equal(uint64(5)))
```

`CallOrder()` returns the names of a fake's methods in the order they were called, for when that order is part of the contract, like a write that has to land before the commit: `Expect(tx.CallOrder()).To(Equal([]string{"Write", "Commit"}))`.

Slice and array arguments are recorded as copies, so a caller that reuses its buffer does not change what the fake recorded; a stub function still receives the original. `fake.Invocations()` returns every recorded call of every method, keyed by method name. Fakes are safe to use from several goroutines at once.

For more examples of using the `counterfeiter` API, look at [some of the provided examples](generated_fakes_test.go). If you are moving from gomock, [docs/migrating-from-gomock.md](docs/migrating-from-gomock.md) goes through the differences.

## Command reference

`go tool counterfeiter -help` prints the following. Outside a module, `go install github.com/aidenwallis/counterfeiter/v7@latest` puts a `counterfeiter` binary in `$GOPATH/bin` that takes the same arguments.

```text
USAGE
	counterfeiter
		[-generate] [-o <output-path>] [-p] [-fake-name <fake-name>]
		[-fake-name-template <template>] [-header <header-file>] [-q] [-test]
		[<source-path>] <interface> [-]

ARGUMENTS
	source-path
		Path to the file or directory containing the interface to fake.
		In package mode (-p), source-path is the import path of the package
		to generate an interface and shim for; a standard library package
		can be given by name (e.g. "os").

	interface
		If source-path is specified: name of the interface to fake.
		If no source-path is specified: fully qualified path of the
		interface to fake, <package-path>.<interface>.
		Not used in package mode (-p), where the interface is named after
		the package.

	example:
		# in directory "mypackage", writes "FakeStdInterface" to
		# ./mypackagefakes/fake_std_interface.go
		counterfeiter package/subpackage.StdInterface

	'-' argument
		Write code to standard out instead of to a file

OPTIONS
	-generate
		Find every //counterfeiter:generate directive in the package in the
		current working directory and in all packages below it, and generate
		fakes for them in one process. Run it from one //go:generate line in
		the package at the root of your module, and it covers the whole
		module:

		# generate.go, at the root of the module
		//go:generate go tool counterfeiter -generate

		Each directive takes the same arguments as the command line,
		relative to the directory of the file it is in. A package with a
		-generate line of its own is left to that line, along with the
		packages below it. Nested modules, testdata and vendor directories,
		and directories starting with "." or "_" are skipped.

	example:
		# in ./store/store.go
		//counterfeiter:generate . Repository
		# in ./api/api.go
		//counterfeiter:generate . Presenter
		//counterfeiter:generate io.WriteCloser

		# run from the module root
		go generate ./...
		# writes "FakeRepository" to ./store/storefakes/fake_repository.go
		# writes "FakePresenter" to ./api/apifakes/fake_presenter.go
		# writes "FakeWriteCloser" to ./api/apifakes/fake_write_closer.go

		The -o, -fake-name-template, -header, -q and -test flags given
		alongside -generate are the defaults for every directive it runs.
		A directive's own flags take precedence. -o is relative to each
		directive's directory; -header is relative to the -generate line.

	example:
		# every package's fakes go into its ./fake and are named after their interface
		//go:generate go tool counterfeiter -generate -o fake -fake-name-template {{.TargetName}}

		# in ./store/store.go
		//counterfeiter:generate . Repository
		//counterfeiter:generate -o otherfake . Cache

		# writes "Repository" to ./store/fake/repository.go
		# writes "Cache" to ./store/otherfake/cache.go

	-o
		Path to the file or directory for the generated fakes.
		This also determines the package name that will be used:
		if the directory already holds a Go package the fake joins
		it, otherwise the package is named after the directory.
		By default, the generated fakes will be generated in
		the package "xyzfakes" which is nested in package "xyz",
		where "xyz" is the name of referenced package.

	example:
		# writes "FakeMyInterface" to ./mySpecialFakesDir/specialFake.go
		counterfeiter -o ./mySpecialFakesDir/specialFake.go ./mypackage MyInterface

		# writes "FakeMyInterface" to ./mySpecialFakesDir/fake_my_interface.go
		counterfeiter -o ./mySpecialFakesDir ./mypackage MyInterface

	-test
		Generate the fake into the external test package ("<package>_test")
		of the output directory, in a _test.go file, so it is only compiled
		for tests and tests in that package can use it unqualified. Without -o the
		fake is written into the current directory, next to the tests that
		use it, wherever the interface comes from. The interface's package
		is imported as usual, so the interface must be exported.
		Cannot be combined with -p.

	example:
		# writes "FakeMyInterface" to ./fake_my_interface_test.go, in package "mypackage_test"
		counterfeiter -test . MyInterface

		# writes "FakeOtherInterface" to ./fake_other_interface_test.go, in package "mypackage_test"
		counterfeiter -test ../otherpackage OtherInterface

		# writes "FakeWriteCloser" to ./fake_write_closer_test.go, in package "mypackage_test"
		counterfeiter -test io.WriteCloser

	-p
		Package mode: counterfeiter generates an interface and a shim
		implementation for a package in your module or the standard
		library. The interface has the package's exported functions as
		methods, and the shim forwards each method to the package. The
		generated file carries a //counterfeiter:generate directive, so
		running go generate there produces a fake of the interface.

	example:
		# writes the "Os" interface and "OsShim" to ${PWD}/osshim/os.go
		counterfeiter -p os
		# now generate "FakeOs" in ${PWD}/osshim/osshimfakes/fake_os.go
		go generate ./osshim/...

	-header
		Path to the file which should be used as a header for all generated fakes.
		By default, no special header is used.
		This is useful to e.g. add a licence header to every fake.

		In generate mode the header can be set once for every directive on
		the "go:generate" line; a "counterfeiter:generate" line that specifies
		its own header file takes precedence.

	example:
		# having the following code in a package ...
		//go:generate go tool counterfeiter -header ./generic.go.txt -generate
		//counterfeiter:generate -header ./specific.go.txt . MyInterface
		//counterfeiter:generate . MyOtherInterface
		//counterfeiter:generate . MyThirdInterface

		# ... generating the fakes ...
		go generate .

		# writes "FakeMyInterface" with ./specific.go.txt as a header
		# writes "FakeMyOtherInterface" & "FakeMyThirdInterface" with ./generic.go.txt as a header

	-fake-name
		Name of the fake struct to generate, used as given. By default,
		'Fake' will be prepended to the name of the original interface;
		a fake of an unexported interface generated into the interface's
		own package is unexported. (ignored in -p mode)

	example:
		# writes "CoolThing" to ./mypackagefakes/cool_thing.go
		counterfeiter -fake-name CoolThing ./mypackage MyInterface

	-fake-name-template
		A text/template for the name of the fake struct, used when -fake-name
		is not given. {{.TargetName}} is the name of the interface being faked,
		with its first letter upper-cased. In generate mode it can be set once
		for every directive on the "go:generate" line. (ignored in -p mode)

	example:
		# writes "MyInterfaceDouble" to ./mypackagefakes/my_interface_double.go
		counterfeiter -fake-name-template '{{.TargetName}}Double' ./mypackage MyInterface

	-q
		Suppress the "Writing ..." status lines. Errors are still reported.
```

## Supported Versions Of `go`

`counterfeiter` follows the [support policy of `go` itself](https://go.dev/doc/devel/release#policy):

> Each major Go release is supported until there are two newer major releases. For example, Go 1.5 was supported until the Go 1.7 release, and Go 1.6 was supported until the Go 1.8 release. We fix critical problems, including [critical security problems](https://go.dev/security), in supported releases as needed by issuing minor revisions (for example, Go 1.6.1, Go 1.6.2, and so on).

If you are having problems with `counterfeiter` and are not using a supported version of go, please update to use a supported version of go before opening an issue.

## Running The Tests For `counterfeiter`

If you want to run the tests for `counterfeiter` (perhaps, because you want to contribute a PR), all you have to do is run `scripts/ci.sh` (`scripts/ci.ps1` on Windows).

## Contributions

So you want to contribute to `counterfeiter`! That's great, here's what to do:

- open a github issue describing your problem or use case, so we can agree on the change before you write it
- write a unit test for the behavior you want, then the simplest code that makes it pass
- look for opportunities to refactor, and keep everything you add covered by tests

`counterfeiter` has a few high level goals for contributors to keep in mind

- keep unit-level test coverage as high as possible
- keep `main.go` as simple as possible
- avoid making the command line options any more complicated
- avoid making the internals of `counterfeiter` any more complicated

If you have any questions about how to contribute, @joefitzgerald maintains `counterfeiter` and will work with you to make it better, together. This project has largely been maintained by the community, and we greatly appreciate any PR (whether big or small).

## License

`counterfeiter` is MIT-licensed.
