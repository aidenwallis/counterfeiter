package command

import (
	"errors"
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func Detect(cwd string, args []string, generateMode bool) ([]Invocation, error) {
	if generateMode {
		return generateModeInvocations(cwd)
	}

	file := os.Getenv("GOFILE")
	var lineno int
	if goline, err := strconv.Atoi(os.Getenv("GOLINE")); err == nil {
		lineno = goline
	}

	i, err := NewInvocation(file, lineno, args)
	if err != nil {
		return nil, err
	}
	i.Dir = cwd
	return []Invocation{i}, nil
}

type Invocation struct {
	Args []string
	Line int
	File string
	// Dir is the directory the invocation runs in: relative paths in its
	// arguments, and the fake's default location, are resolved against it.
	Dir string
}

func NewInvocation(file string, line int, args []string) (Invocation, error) {
	if len(args) < 1 {
		return Invocation{}, fmt.Errorf("%s:%v an invocation of counterfeiter must have arguments", file, line)
	}
	i := Invocation{
		File: file,
		Line: line,
		Args: args,
	}
	return i, nil
}

// generateModeInvocations collects the '//counterfeiter:generate' directives
// of the package in cwd and of every package below it, down to the next
// package with a '-generate' line of its own, which go generate runs
// separately. Nested modules, and directories the go command ignores, are
// skipped. Each directive runs in the directory of the file it is in.
func generateModeInvocations(cwd string) ([]Invocation, error) {
	var result []Invocation
	err := filepath.WalkDir(cwd, func(dir string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if dir != cwd && ignoredDir(dir, d.Name()) {
			return filepath.SkipDir
		}
		invocations, ownGenerateLine, err := invocationsInDir(dir)
		if err != nil {
			return err
		}
		if dir != cwd && ownGenerateLine {
			return filepath.SkipDir
		}
		result = append(result, invocations...)
		return nil
	})
	return result, err
}

func ignoredDir(dir, name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "vendor" {
		return true
	}
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// invocationsInDir returns the directives in the package in dir, and whether
// the package has a '//go:generate' line running counterfeiter -generate.
func invocationsInDir(dir string) ([]Invocation, bool, error) {
	pkg, err := build.ImportDir(dir, build.IgnoreVendor)
	var noGo *build.NoGoError
	if errors.As(err, &noGo) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	gofiles := make([]string, 0, len(pkg.GoFiles)+len(pkg.CgoFiles)+len(pkg.TestGoFiles)+len(pkg.XTestGoFiles))
	gofiles = append(gofiles, pkg.GoFiles...)
	gofiles = append(gofiles, pkg.CgoFiles...)
	gofiles = append(gofiles, pkg.TestGoFiles...)
	gofiles = append(gofiles, pkg.XTestGoFiles...)
	sort.Strings(gofiles)

	var result []Invocation
	ownGenerateLine := false
	for _, file := range gofiles {
		invocations, generateLine, err := invocationsInFile(dir, file)
		if err != nil {
			return nil, false, err
		}
		result = append(result, invocations...)
		ownGenerateLine = ownGenerateLine || generateLine
	}
	return result, ownGenerateLine, nil
}

func invocationsInFile(dir string, file string) ([]Invocation, bool, error) {
	str, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return nil, false, err
	}
	lines := strings.Split(string(str), "\n")

	var result []Invocation
	generateLine := false
	line := 0
	for i := range lines {
		line++
		generateLine = generateLine || isGenerateLine(lines[i])
		args, ok := matchForString(lines[i])
		if !ok {
			continue
		}
		inv, err := NewInvocation(file, line, args)
		if err != nil {
			return nil, false, err
		}
		inv.Dir = dir

		result = append(result, inv)
	}

	return result, generateLine, nil
}

// isGenerateLine reports whether s is a '//go:generate' line that runs
// counterfeiter in generate mode.
func isGenerateLine(s string) bool {
	fields := strings.Fields(strings.TrimSuffix(s, "\r"))
	return len(fields) > 0 && fields[0] == "//go:generate" &&
		slices.ContainsFunc(fields, func(f string) bool { return strings.Contains(f, "counterfeiter") }) &&
		slices.Contains(fields, "-generate")
}

const generateDirectivePrefix = "//counterfeiter:generate "

func matchForString(s string) ([]string, bool) {
	if !strings.HasPrefix(s, generateDirectivePrefix) {
		return nil, false
	}
	return stringToArgs(s[len(generateDirectivePrefix):]), true
}

func stringToArgs(s string) []string {
	a := strings.Fields(s)
	result := []string{
		"counterfeiter",
	}
	result = append(result, a...)
	return result
}
