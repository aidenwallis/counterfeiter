package command

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// configNames are the names counterfeiter looks for, in this order, when it
// is run without arguments.
var configNames = []string{".counterfeiter.yaml", ".counterfeiter.yml"}

// config is the content of a .counterfeiter.yaml file. Its options mean what
// the flags of the same name mean.
type config struct {
	options    `yaml:",inline"`
	Directives []string                  `yaml:"directives"`
	Packages   map[string]*packageConfig `yaml:"packages"`
}

type packageConfig struct {
	options    `yaml:",inline"`
	Interfaces map[string]*interfaceConfig `yaml:"interfaces"`
}

type interfaceConfig struct {
	options  `yaml:",inline"`
	FakeName string `yaml:"fake-name"`
}

type options struct {
	Output           string `yaml:"o"`
	Header           string `yaml:"header"`
	FakeNameTemplate string `yaml:"fake-name-template"`
	Test             *bool  `yaml:"test"`
	Quiet            *bool  `yaml:"q"`
}

// over returns o with every option that more sets replaced by its value.
func (o options) over(more options) options {
	if more.Output != "" {
		o.Output = more.Output
	}
	if more.Header != "" {
		o.Header = more.Header
	}
	if more.FakeNameTemplate != "" {
		o.FakeNameTemplate = more.FakeNameTemplate
	}
	if more.Test != nil {
		o.Test = more.Test
	}
	if more.Quiet != nil {
		o.Quiet = more.Quiet
	}
	return o
}

func (o options) flags(fakeName string) []string {
	var result []string
	if o.Output != "" {
		result = append(result, "-o", o.Output)
	}
	if o.Header != "" {
		result = append(result, "-header", o.Header)
	}
	if fakeName != "" {
		result = append(result, "-fake-name", fakeName)
	}
	if o.FakeNameTemplate != "" {
		result = append(result, "-fake-name-template", o.FakeNameTemplate)
	}
	if o.Test != nil && *o.Test {
		result = append(result, "-test")
	}
	if o.Quiet != nil && *o.Quiet {
		result = append(result, "-q")
	}
	return result
}

// findConfig returns the config file in dir or the closest directory above
// it, without leaving the module dir belongs to, and the directory of that
// module's go.mod. It returns "" for the file when there is none.
func findConfig(dir string) (file string, moduleDir string) {
	for {
		if file == "" {
			for _, name := range configNames {
				if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
					file = filepath.Join(dir, name)
					break
				}
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return file, dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return file, ""
		}
		dir = parent
	}
}

func configInvocations(file, moduleDir string) ([]Invocation, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var c config
	d := yaml.NewDecoder(bytes.NewReader(b))
	d.KnownFields(true)
	if err := d.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %v", file, err)
	}
	dir := filepath.Dir(file)
	modulePath := ""
	if moduleDir != "" {
		if gomod, err := os.ReadFile(filepath.Join(moduleDir, "go.mod")); err == nil {
			modulePath = modfile.ModulePath(gomod)
		}
	}
	resolveHeader(&c.options, dir)

	var result []Invocation
	for _, key := range sortedKeys(c.Packages) {
		p := c.Packages[key]
		if p == nil || len(p.Interfaces) == 0 {
			return nil, fmt.Errorf("%s: package %s lists no interfaces", file, key)
		}
		resolveHeader(&p.options, dir)
		pkgDir := localDir(key, dir, moduleDir, modulePath)
		for _, name := range sortedKeys(p.Interfaces) {
			i := p.Interfaces[name]
			if i == nil {
				i = &interfaceConfig{}
			}
			resolveHeader(&i.options, dir)
			o := c.options.over(p.options).over(i.options)
			args := append([]string{"counterfeiter"}, o.flags(i.FakeName)...)
			inv := Invocation{File: file, Dir: pkgDir}
			if pkgDir != "" {
				inv.Args = append(args, ".", name)
			} else {
				if o.Output == "" {
					return nil, fmt.Errorf("%s: package %s is not in this module, so it needs an o to say where its fakes go", file, key)
				}
				inv.Dir = dir
				inv.Args = append(args, key+"."+name)
			}
			result = append(result, inv)
		}
	}

	directives, err := directiveInvocations(dir, c.Directives, c.options.flags(""))
	if err != nil {
		return nil, err
	}
	return append(result, directives...), nil
}

// localDir returns the directory of the package key names if it is in the
// module, or "" if it is not.
func localDir(key, dir, moduleDir, modulePath string) string {
	if filepath.IsAbs(key) {
		return key
	}
	if key == "." || key == ".." || strings.HasPrefix(key, "./") || strings.HasPrefix(key, "../") {
		return filepath.Join(dir, filepath.FromSlash(key))
	}
	if modulePath == "" {
		return ""
	}
	if key == modulePath {
		return moduleDir
	}
	if rest, ok := strings.CutPrefix(key, modulePath+"/"); ok {
		return filepath.Join(moduleDir, filepath.FromSlash(path.Clean(rest)))
	}
	return ""
}

// directiveInvocations finds the packages that match patterns and returns the
// invocations for their //counterfeiter:generate directives, with defaults
// ahead of each directive's own flags so that those take precedence.
func directiveInvocations(dir string, patterns []string, defaults []string) ([]Invocation, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedFiles, Dir: dir}, patterns...)
	if err != nil {
		return nil, err
	}
	var dirs []string
	seen := map[string]bool{}
	for _, p := range pkgs {
		if p.Dir != "" && !seen[p.Dir] {
			seen[p.Dir] = true
			dirs = append(dirs, p.Dir)
		}
	}
	sort.Strings(dirs)
	var result []Invocation
	for _, d := range dirs {
		invocations, err := generateModeInvocations(d)
		if err != nil {
			return nil, err
		}
		for _, inv := range invocations {
			inv.Args = append(append([]string{inv.Args[0]}, defaults...), inv.Args[1:]...)
			result = append(result, inv)
		}
	}
	return result, nil
}

func resolveHeader(o *options, dir string) {
	if o.Header != "" && !filepath.IsAbs(o.Header) {
		o.Header = filepath.Join(dir, o.Header)
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
