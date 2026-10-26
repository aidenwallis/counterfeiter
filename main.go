package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"slices"
	"sync"

	"github.com/aidenwallis/counterfeiter/v7/arguments"
	"github.com/aidenwallis/counterfeiter/v7/command"
	"github.com/aidenwallis/counterfeiter/v7/generator"
)

func main() {
	debug.SetGCPercent(-1)

	if err := run(); err != nil {
		fail("%v", err)
	}
}

func run() error {
	profile := os.Getenv("COUNTERFEITER_PROFILE") != ""
	if profile {
		p, err := filepath.Abs(filepath.Join(".", "counterfeiter.profile"))
		if err != nil {
			return err
		}
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}
		fmt.Printf("Profile: %s\n", p)
		defer pprof.StopCPUProfile()
	}

	log.SetFlags(log.Lshortfile)
	if !isDebug() {
		log.SetOutput(io.Discard)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return errors.New("Error - couldn't determine current working directory")
	}

	var cache generator.Cacher
	var headerReader generator.FileReader
	if disableCache() {
		cache = &generator.FakeCache{}
		headerReader = &generator.SimpleFileReader{}
	} else {
		cache = &generator.Cache{}
		headerReader = &generator.CachedFileReader{}
	}
	var invocations []command.Invocation
	var args *arguments.ParsedArguments
	args, _ = arguments.New(os.Args, cwd, filepath.EvalSymlinks, os.Stat)
	generateMode := false
	if args != nil {
		generateMode = args.GenerateMode
	}
	if !generateMode && shouldPrintGenerateWarning() {
		fmt.Printf("\nWARNING: Invoking counterfeiter multiple times from \"go generate\" is slow.\nConsider using counterfeiter:generate directives to speed things up.\nSee https://github.com/aidenwallis/counterfeiter#step-2---add-directives for more information.\nSet the \"COUNTERFEITER_NO_GENERATE_WARNING\" environment variable to suppress this message.\n\n")
	}
	invocations, err = command.Detect(cwd, os.Args, generateMode)
	if err != nil {
		return err
	}

	parsed := make([]*arguments.ParsedArguments, len(invocations))
	var targets []string
	for i, inv := range invocations {
		// Flags on the '//go:generate ... -generate' line are the defaults for
		// every '//counterfeiter:generate' directive in the package.
		parsed[i], err = arguments.New(inv.Args, inv.Dir, filepath.EvalSymlinks, os.Stat, arguments.WithDefaults(args))
		if err != nil {
			return err
		}
		if !slices.Contains(targets, parsed[i].PackagePath) {
			targets = append(targets, parsed[i].PackagePath)
		}
	}
	// One load for every target costs about as much as a load for one.
	if len(targets) > 1 && !disableCache() {
		if err := generator.Preload(cache, cwd, targets...); err != nil {
			return err
		}
	}

	// Fakes are built in parallel but written in order, so the output and
	// the files written before a failure are those of a sequential run.
	for i, f := range buildAll(invocations, parsed, cache, headerReader) {
		if err := write(cwd, parsed[i], f); err != nil {
			return err
		}
	}
	return nil
}

func isDebug() bool {
	return os.Getenv("COUNTERFEITER_DEBUG") != ""
}

func disableCache() bool {
	return os.Getenv("COUNTERFEITER_DISABLECACHE") != ""
}

func shouldPrintGenerateWarning() bool {
	return invokedByGoGenerate() && os.Getenv("COUNTERFEITER_NO_GENERATE_WARNING") == ""
}

func invokedByGoGenerate() bool {
	return os.Getenv("DOLLAR") == "$"
}

// fake is a generated fake, or the reason it could not be generated.
type fake struct {
	name string
	code []byte
	err  error
}

func buildAll(invocations []command.Invocation, parsed []*arguments.ParsedArguments, cache generator.Cacher, headerReader generator.FileReader) []fake {
	fakes := make([]fake, len(invocations))
	var wg sync.WaitGroup
	limit := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i := range invocations {
		wg.Add(1)
		limit <- struct{}{}
		go func() {
			defer func() { <-limit; wg.Done() }()
			fakes[i] = build(invocations[i].Dir, parsed[i], cache, headerReader)
		}()
	}
	wg.Wait()
	return fakes
}

func build(workingDir string, args *arguments.ParsedArguments, cache generator.Cacher, headerReader generator.FileReader) fake {
	f, err := newFake(workingDir, args, cache, headerReader)
	if err != nil {
		return fake{err: err}
	}
	b, err := f.Generate(true)
	return fake{name: f.Name, code: b, err: err}
}

func write(workingDir string, args *arguments.ParsedArguments, f fake) error {
	if f.name != "" && !args.Quiet {
		if err := reportStarting(workingDir, args.OutputPath, f.name); err != nil {
			return err
		}
	}
	if f.err != nil {
		return f.err
	}

	if err := printCode(f.code, args.OutputPath, args.PrintToStdOut); err != nil {
		return err
	}

	if !args.Quiet {
		fmt.Fprint(os.Stderr, "Done\n")
	}

	return nil
}

func newFake(workingDir string, args *arguments.ParsedArguments, cache generator.Cacher, headerReader generator.FileReader) (*generator.Fake, error) {
	mode := generator.InterfaceOrFunction
	if args.GenerateInterfaceAndShimFromPackageDirectory {
		mode = generator.Package
	}

	headerContent, err := headerReader.Get(workingDir, args.HeaderFile)
	if err != nil {
		return nil, err
	}

	opts := []generator.Option{generator.WithDestinationDir(filepath.Dir(args.OutputPath))}
	if args.TestPackage {
		opts = append(opts, generator.WithTestPackage())
	}
	if args.FakeNameExplicit {
		opts = append(opts, generator.WithExplicitName())
	}
	return generator.NewFake(mode, args.InterfaceName, args.PackagePath, args.FakeImplName, args.DestinationPackageName, headerContent, workingDir, cache, opts...)
}

func printCode(formattedCode []byte, outputPath string, printToStdOut bool) error {
	if printToStdOut {
		fmt.Println(string(formattedCode))
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(outputPath), 0777)
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Couldn't create fake file - %v", err)
	}

	_, err = file.Write(formattedCode)
	if err != nil {
		return fmt.Errorf("Couldn't write to fake file - %v", err)
	}
	return nil
}

func reportStarting(workingDir string, outputPath, fakeName string) error {
	rel, err := filepath.Rel(workingDir, outputPath)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Writing `%s` to `%s`... ", fakeName, rel)
	if isDebug() {
		msg = msg + "\n"
	}
	fmt.Fprint(os.Stderr, msg)
	return nil
}

func fail(s string, args ...interface{}) {
	fmt.Printf("\n"+s+"\n", args...)
	os.Exit(1)
}
