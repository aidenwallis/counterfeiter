package command_test

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/aidenwallis/counterfeiter/v7/command"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestRunner(t *testing.T) {
	spec.Run(t, "Runner", testRunner, spec.Report(report.Terminal{}))
}

func testRunner(t *testing.T, when spec.G, it spec.S) {
	reset := func() {
		os.Unsetenv("DOLLAR")
		os.Unsetenv("GOFILE")
		os.Unsetenv("GOLINE")
		os.Unsetenv("GOPACKAGE")
	}

	it.Before(func() {
		RegisterTestingT(t)
		reset()
		log.SetFlags(log.Llongfile)
	})

	it.After(func() {
		reset()
	})

	when("counterfeiter has been invoked directly", func() {
		it.Before(func() {
		})

		it("creates an invocation", func() {
			i, err := command.Detect(filepath.Join(".", "..", "fixtures"), []string{"counterfeiter", ".", "AliasedInterface"}, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(i).NotTo(BeNil())
			Expect(i).To(HaveLen(1))
			Expect(i[0].Args).To(HaveLen(3))
			Expect(i[0].Args[1]).To(Equal("."))
			Expect(i[0].Args[2]).To(Equal("AliasedInterface"))
		})
	})

	when("counterfeiter is invoked in generate mode", func() {
		it.Before(func() {
			os.Unsetenv("DOLLAR")
			os.Unsetenv("GOFILE")
			os.Unsetenv("GOLINE")
			os.Unsetenv("GOPACKAGE")
		})

		it("creates invocations", func() {
			i, err := command.Detect(filepath.Join(".", "..", "fixtures"), []string{"counterfeiter", ".", "AliasedInterface"}, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(i).NotTo(BeNil())
			Expect(len(i)).To(Equal(21))
			Expect(i[0].File).To(Equal("aliased_interfaces.go"))
			Expect(i[0].Line).To(Equal(6))
			Expect(i[0].Args).To(HaveLen(3))
			Expect(i[0].Args[0]).To(Equal("counterfeiter"))
			Expect(i[0].Args[1]).To(Equal("."))
			Expect(i[0].Args[2]).To(Equal("AliasedInterface"))
		})
	})

	when("counterfeiter is invoked in generate mode above other packages", func() {
		var root string

		write := func(rel, content string) {
			p := filepath.Join(root, filepath.FromSlash(rel))
			Expect(os.MkdirAll(filepath.Dir(p), 0o777)).To(Succeed())
			Expect(os.WriteFile(p, []byte(content), 0o666)).To(Succeed())
		}

		it.Before(func() {
			root = t.TempDir()
			write("go.mod", "module example.com/root\n")
			write("root.go", "package root\n\n//go:generate go tool counterfeiter -generate\n\n//counterfeiter:generate . Root\n")
			write("api/api.go", "package api\n\n//counterfeiter:generate . API\n")
			write("api/v2/v2.go", "package v2\n\n//counterfeiter:generate . V2\n")
			write("empty/README.md", "//counterfeiter:generate . Ignored\n")
			write("owned/owned.go", "package owned\n\n//go:generate go run github.com/aidenwallis/counterfeiter/v7 -q -generate\n\n//counterfeiter:generate . Owned\n")
			write("owned/below/below.go", "package below\n\n//counterfeiter:generate . Below\n")
			write("nested/go.mod", "module example.com/nested\n")
			write("nested/nested.go", "package nested\n\n//counterfeiter:generate . Nested\n")
			write("testdata/td.go", "package td\n\n//counterfeiter:generate . TestData\n")
			write("vendor/v/v.go", "package v\n\n//counterfeiter:generate . Vendored\n")
			write(".hidden/h.go", "package h\n\n//counterfeiter:generate . Hidden\n")
			write("_skip/s.go", "package s\n\n//counterfeiter:generate . Skipped\n")
		})

		it("runs the directives of every package below it that has no -generate line of its own", func() {
			i, err := command.Detect(root, []string{"counterfeiter", "-generate"}, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(3))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter", ".", "Root"}))
			Expect(i[0].Dir).To(Equal(root))
			Expect(i[1].Args).To(Equal([]string{"counterfeiter", ".", "API"}))
			Expect(i[1].Dir).To(Equal(filepath.Join(root, "api")))
			Expect(i[1].File).To(Equal("api.go"))
			Expect(i[1].Line).To(Equal(3))
			Expect(i[2].Args).To(Equal([]string{"counterfeiter", ".", "V2"}))
			Expect(i[2].Dir).To(Equal(filepath.Join(root, "api", "v2")))
		})

		it("leaves a package with its own -generate line, and everything below it, to that line", func() {
			i, err := command.Detect(filepath.Join(root, "owned"), []string{"counterfeiter", "-generate"}, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(2))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter", ".", "Owned"}))
			Expect(i[1].Args).To(Equal([]string{"counterfeiter", ".", "Below"}))
			Expect(i[1].Dir).To(Equal(filepath.Join(root, "owned", "below")))
		})
	})

	when("counterfeiter has been invoked by go generate", func() {
		it.Before(func() {
			os.Setenv("DOLLAR", "$")
			os.Setenv("GOFILE", "aliased_interfaces.go")
			os.Setenv("GOLINE", "5")
			os.Setenv("GOPACKAGE", "fixtures")
		})

		it("creates invocations but does not include generate mode as an invocation", func() {
			i, err := command.Detect(filepath.Join(".", "..", "fixtures"), []string{"counterfeiter", ".", "AliasedInterface"}, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(i).NotTo(BeNil())
			Expect(len(i)).To(Equal(1))
			Expect(i[0].File).To(Equal("aliased_interfaces.go"))
			Expect(i[0].Line).To(Equal(5))
			Expect(i[0].Args).To(HaveLen(3))
			Expect(i[0].Args[0]).To(Equal("counterfeiter"))
			Expect(i[0].Args[1]).To(Equal("."))
			Expect(i[0].Args[2]).To(Equal("AliasedInterface"))
		})
	})
}
