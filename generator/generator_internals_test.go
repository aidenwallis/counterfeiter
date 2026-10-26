package generator

import (
	"go/types"
	"io"
	"log"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"golang.org/x/tools/go/packages"
)

func TestGenerator(t *testing.T) {
	log.SetOutput(io.Discard) // Comment this out to see verbose log output
	log.SetFlags(log.Llongfile)
	spec.Run(t, "Generator", testGenerator, spec.Report(report.Terminal{}), spec.Parallel())
}

func testGenerator(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)
	var (
		f   *Fake
		err error
	)

	when("constructing a fake with NewFake()", func() {
		when("the target is a nonexistent package", func() {
			it("errors", func() {
				c := &Cache{}
				f, err = NewFake(InterfaceOrFunction, "NonExistent", "nonexistentpackage", "FakeNonExistent", "nonexistentpackagefakes", "", "", c)
				g.Expect(err).To(HaveOccurred())
				g.Expect(f).To(BeNil())
			})
		})

		when("the target is a package with a nonexistent interface", func() {
			it("errors", func() {
				c := &Cache{}
				f, err = NewFake(InterfaceOrFunction, "NonExistent", "os", "FakeNonExistent", "osfakes", "", "", c)
				g.Expect(err).To(HaveOccurred())
				g.Expect(f).To(BeNil())
			})
		})

		when("the target is an interface that exists", func() {
			it("succeeds", func() {
				c := &Cache{}
				f, err = NewFake(InterfaceOrFunction, "FileInfo", "os", "FakeFileInfo", "osfakes", "", "", c)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f).NotTo(BeNil())
				g.Expect(f.TargetAlias).To(Equal("os"))
				g.Expect(f.TargetName).To(Equal("FileInfo"))
				g.Expect(f.TargetPackage).To(Equal("os"))
				g.Expect(f.Name).To(Equal("FakeFileInfo"))
				g.Expect(f.Mode).To(Equal(InterfaceOrFunction))
				g.Expect(f.DestinationPackage).To(Equal("osfakes"))
				g.Expect(f.Imports).To(BeEquivalentTo(Imports{
					ByAlias: map[string]Import{
						"os":   {Alias: "os", PkgPath: "os"},
						"sync": {Alias: "sync", PkgPath: "sync"},
						"time": {Alias: "time", PkgPath: "time"},
						"fs":   {Alias: "fs", PkgPath: "io/fs"},
					},
					ByPkgPath: map[string]Import{
						"os":    {Alias: "os", PkgPath: "os"},
						"sync":  {Alias: "sync", PkgPath: "sync"},
						"time":  {Alias: "time", PkgPath: "time"},
						"io/fs": {Alias: "fs", PkgPath: "io/fs"},
					},
				}))
				g.Expect(f.Function).To(BeZero())
				g.Expect(f.Packages).NotTo(BeNil())
				g.Expect(f.Package).NotTo(BeNil())
				g.Expect(f.Methods).To(HaveLen(6))
			})
		})

		when("the target is an interface in a third-party module", func() {
			it("succeeds", func() {
				c := &Cache{}
				f, err = NewFake(InterfaceOrFunction, "GomegaMatcher", "github.com/onsi/gomega/types", "FakeGomegaMatcher", "typesfakes", "", "", c)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.TargetPackage).To(Equal("github.com/onsi/gomega/types"))
				b, err := f.Generate(true)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(b)).To(ContainSubstring(`"github.com/onsi/gomega/types"`))
				g.Expect(string(b)).To(ContainSubstring("func (fake *FakeGomegaMatcher) Match(arg1 any) (bool, error)"))
				g.Expect(string(b)).To(ContainSubstring("var _ types.GomegaMatcher = new(FakeGomegaMatcher)"))
			})
		})

		when("the target is an unexported type that nothing else in its package refers to", func() {
			it("still finds it", func() {
				c := &Cache{}
				for _, target := range []string{"unexportedInterface", "unexportedFunc"} {
					f, err = NewFake(InterfaceOrFunction, target, "github.com/aidenwallis/counterfeiter/v7/fixtures", "Fake"+target, "fixturesfakes", "", "", c)
					g.Expect(err).NotTo(HaveOccurred(), target)
					g.Expect(f.TargetName).To(Equal(target))
				}
			})
		})

		when("the target is a generic interface whose constraint comes from another package", func() {
			it("imports the constraint's package and qualifies it with that package's name", func() {
				c := &Cache{}
				f, err = NewFake(InterfaceOrFunction, "GenericImportedConstraint", "github.com/aidenwallis/counterfeiter/v7/fixtures", "FakeGenericImportedConstraint", "fixturesfakes", "", "", c)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.Imports.ByPkgPath).To(HaveKey("github.com/aidenwallis/counterfeiter/v7/fixtures/go-hyphenpackage"))
				g.Expect(f.GenericTypeParametersAndConstraints).To(Equal("[T hyphenpackage.Hyphenated]"))
				g.Expect(f.GenericTypeParameters).To(Equal("[T]"))
			})
		})

		when("the target uses a package whose name is the major version at the end of its path", func() {
			it("names that import itself, so goimports does not have to look the package up", func() {
				f, err = NewFake(InterfaceOrFunction, "ImportsVersionedPackage", "github.com/aidenwallis/counterfeiter/v7/fixtures", "FakeImportsVersionedPackage", "fixturesfakes", "", "", &Cache{})
				g.Expect(err).NotTo(HaveOccurred())
				b, err := f.Generate(false)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(b)).To(ContainSubstring(`v1 "github.com/aidenwallis/counterfeiter/v7/fixtures/versioned/v1"`))
			})
		})

		when("the target is an unexported interface faked outside its package", func() {
			it("does not import that package, which the fake cannot name", func() {
				f, err = NewFake(InterfaceOrFunction, "unexportedInterface", "github.com/aidenwallis/counterfeiter/v7/fixtures", "FakeUnexportedInterface", "fixturesfakes", "", "", &Cache{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.Imports.ByPkgPath).NotTo(HaveKey("github.com/aidenwallis/counterfeiter/v7/fixtures"))
			})
		})

		when("the target is a package", func() {
			it("does not import sync, which a shim does not use", func() {
				f, err = NewFake(Package, "", "github.com/aidenwallis/counterfeiter/v7/fixtures/packagemode", "Packagemode", "packagemodeshim", "", "", &Cache{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.Imports.ByPkgPath).NotTo(HaveKey("sync"))
			})
		})

		when("the target package has tests", func() {
			it("loads the package alone, without the test variants that would type-check it again", func() {
				f, err = NewFake(InterfaceOrFunction, "Sower", "github.com/aidenwallis/counterfeiter/v7/fixtures/seeded", "FakeSower", "fakes", "", "", &Cache{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.Packages).To(HaveLen(1))
			})

			it("still finds a target declared in a _test.go file", func() {
				f, err = NewFake(InterfaceOrFunction, "DeclaredInTest", "github.com/aidenwallis/counterfeiter/v7/fixtures/testonly", "FakeDeclaredInTest", "testonlyfakes", "", "", &Cache{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.IsInterface()).To(BeTrue())
			})
		})

		when("the destination directory already holds a package", func() {
			it("joins that package rather than naming one after the directory", func() {
				dir, err := filepath.Abs(filepath.Join("..", "fixtures", "seeded", "fakes"))
				g.Expect(err).NotTo(HaveOccurred())
				f, err = NewFake(InterfaceOrFunction, "Sower", "github.com/aidenwallis/counterfeiter/v7/fixtures/seeded", "FakeSower", "fakes", "", "", &Cache{}, WithDestinationDir(dir))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.DestinationPackage).To(Equal("seeded_fakes"))
				b, err := f.Generate(false)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(b)).To(ContainSubstring("package seeded_fakes\n"))
				g.Expect(string(b)).To(ContainSubstring("var _ seeded.Sower = new(FakeSower)"))
			})

			it("keeps the given name when the directory holds no Go files", func() {
				f, err = NewFake(InterfaceOrFunction, "Sower", "github.com/aidenwallis/counterfeiter/v7/fixtures/seeded", "FakeSower", "fakes", "", "", &Cache{}, WithDestinationDir(t.TempDir()))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.DestinationPackage).To(Equal("fakes"))
			})
		})

		when("the destination is the same package as the target", func() {
			var (
				pkgPath string
				dir     string
			)

			it.Before(func() {
				pkgPath = "github.com/aidenwallis/counterfeiter/v7/fixtures/samepackage"
				var err error
				dir, err = filepath.Abs(filepath.Join("..", "fixtures", "samepackage"))
				g.Expect(err).NotTo(HaveOccurred())
			})

			it("does not import the target package and leaves its types unqualified", func() {
				f, err = NewFake(InterfaceOrFunction, "Widget", pkgPath, "FakeWidget", "samepackage", "", "", &Cache{}, WithDestinationDir(dir))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.TargetAlias).To(BeEmpty())
				g.Expect(f.Imports.ByPkgPath).NotTo(HaveKey(pkgPath))
				b, err := f.Generate(true)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(b)).To(ContainSubstring("Do(arg1 Thing) (Thing, error)"))
				g.Expect(string(b)).To(ContainSubstring("var _ Widget = new(FakeWidget)"))
				g.Expect(string(b)).NotTo(ContainSubstring("samepackage."))
			})

			it("keeps the fake of an unexported interface unexported and asserts that it implements it", func() {
				f, err = NewFake(InterfaceOrFunction, "gadget", pkgPath, "FakeGadget", "samepackage", "", "", &Cache{}, WithDestinationDir(dir))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.Name).To(Equal("fakeGadget"))
				b, err := f.Generate(false)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(b)).To(ContainSubstring("type fakeGadget struct"))
				g.Expect(string(b)).To(ContainSubstring("var _ gadget = new(fakeGadget)"))
				g.Expect(string(b)).NotTo(ContainSubstring("FakeGadget"))
			})

			it("uses a name given with -fake-name as it is, even for an unexported interface", func() {
				f, err = NewFake(InterfaceOrFunction, "gadget", pkgPath, "GadgetDouble", "samepackage", "", "", &Cache{}, WithDestinationDir(dir), WithExplicitName())
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.Name).To(Equal("GadgetDouble"))
				b, err := f.Generate(false)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(b)).To(ContainSubstring("var _ gadget = new(GadgetDouble)"))
			})

			it("names the package after the target's package, not its directory", func() {
				hyphenDir, err := filepath.Abs(filepath.Join("..", "fixtures", "go-hyphenpackage"))
				g.Expect(err).NotTo(HaveOccurred())
				f, err = NewFake(InterfaceOrFunction, "Hyphenated", "github.com/aidenwallis/counterfeiter/v7/fixtures/go-hyphenpackage", "FakeHyphenated", "gohyphenpackage", "", "", &Cache{}, WithDestinationDir(hyphenDir))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(f.DestinationPackage).To(Equal("hyphenpackage"))
				b, err := f.Generate(false)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(b)).To(ContainSubstring("package hyphenpackage\n"))
			})

			when("the fake goes into the external test package", func() {
				it("names the package <package>_test and imports the target package", func() {
					f, err = NewFake(InterfaceOrFunction, "Widget", pkgPath, "FakeWidget", "samepackage_test", "", "", &Cache{}, WithDestinationDir(dir), WithTestPackage())
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(f.DestinationPackage).To(Equal("samepackage_test"))
					g.Expect(f.TargetAlias).To(Equal("samepackage"))
					g.Expect(f.Imports.ByPkgPath).To(HaveKey(pkgPath))
					b, err := f.Generate(true)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(string(b)).To(ContainSubstring("package samepackage_test\n"))
					g.Expect(string(b)).To(ContainSubstring("Do(arg1 samepackage.Thing) (samepackage.Thing, error)"))
					g.Expect(string(b)).To(ContainSubstring("var _ samepackage.Widget = new(FakeWidget)"))
				})

				it("refuses an unexported interface, which the test package cannot see", func() {
					f, err = NewFake(InterfaceOrFunction, "gadget", pkgPath, "FakeGadget", "samepackage_test", "", "", &Cache{}, WithDestinationDir(dir), WithTestPackage())
					g.Expect(err).To(MatchError(And(ContainSubstring("gadget"), ContainSubstring("samepackage_test"), ContainSubstring("unexported"))))
				})

				it("uses the name of the package in the destination directory, not the directory name", func() {
					hyphenDir, err := filepath.Abs(filepath.Join("..", "fixtures", "go-hyphenpackage"))
					g.Expect(err).NotTo(HaveOccurred())
					f, err = NewFake(InterfaceOrFunction, "WriteCloser", "io", "FakeWriteCloser", "gohyphenpackage_test", "", "", &Cache{}, WithDestinationDir(hyphenDir), WithTestPackage())
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(f.DestinationPackage).To(Equal("hyphenpackage_test"))
					g.Expect(f.TargetAlias).To(Equal("io"))
					g.Expect(f.Imports.ByPkgPath).To(HaveKey("io"))
				})

				it("keeps the given name when the destination directory has no Go files", func() {
					f, err = NewFake(InterfaceOrFunction, "Widget", pkgPath, "FakeWidget", "other_test", "", "", &Cache{}, WithDestinationDir(t.TempDir()), WithTestPackage())
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(f.DestinationPackage).To(Equal("other_test"))
					g.Expect(f.TargetAlias).To(Equal("samepackage"))
				})

				it("fakes an interface from a third-party module into the test package", func() {
					testDir, err := filepath.Abs(filepath.Join("..", "fixtures", "externaltest"))
					g.Expect(err).NotTo(HaveOccurred())
					f, err = NewFake(InterfaceOrFunction, "GomegaMatcher", "github.com/onsi/gomega/types", "FakeGomegaMatcher", "externaltest_test", "", "", &Cache{}, WithDestinationDir(testDir), WithTestPackage())
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(f.DestinationPackage).To(Equal("externaltest_test"))
					g.Expect(f.TargetAlias).To(Equal("types"))
					g.Expect(f.Imports.ByPkgPath).To(HaveKey("github.com/onsi/gomega/types"))
					b, err := f.Generate(true)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(string(b)).To(ContainSubstring("package externaltest_test\n"))
					g.Expect(string(b)).To(ContainSubstring("var _ types.GomegaMatcher = new(FakeGomegaMatcher)"))
				})
			})

			when("the destination only shares the target's package name", func() {
				it("still imports the target package", func() {
					other := t.TempDir()
					f, err = NewFake(InterfaceOrFunction, "Widget", pkgPath, "FakeWidget", "samepackage", "", "", &Cache{}, WithDestinationDir(other))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(f.TargetAlias).To(Equal("samepackage"))
					g.Expect(f.Imports.ByPkgPath).To(HaveKey(pkgPath))
					b, err := f.Generate(false)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(string(b)).To(ContainSubstring("var _ samepackage.Widget = new(FakeWidget)"))
				})

				it("does not assert an unexported interface, and keeps the fake exported", func() {
					other := t.TempDir()
					f, err = NewFake(InterfaceOrFunction, "gadget", pkgPath, "FakeGadget", "samepackage", "", "", &Cache{}, WithDestinationDir(other))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(f.Name).To(Equal("FakeGadget"))
					b, err := f.Generate(false)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(string(b)).NotTo(ContainSubstring("var _ "))
				})
			})
		})

		when("the target is a function that exists", func() {
			it("succeeds", func() {
				c := &Cache{}
				f, err = NewFake(InterfaceOrFunction, "HandlerFunc", "net/http", "FakeHandlerFunc", "httpfakes", "", "", c)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(f).NotTo(BeNil())
				g.Expect(f.TargetAlias).To(Equal("http"))
				g.Expect(f.TargetName).To(Equal("HandlerFunc"))
				g.Expect(f.TargetPackage).To(Equal("net/http"))
				g.Expect(f.Name).To(Equal("FakeHandlerFunc"))
				g.Expect(f.Mode).To(Equal(InterfaceOrFunction))
				g.Expect(f.DestinationPackage).To(Equal("httpfakes"))
				g.Expect(f.Imports).To(BeEquivalentTo(Imports{
					ByAlias: map[string]Import{
						"http": {Alias: "http", PkgPath: "net/http"},
						"sync": {Alias: "sync", PkgPath: "sync"},
					},
					ByPkgPath: map[string]Import{
						"net/http": {Alias: "http", PkgPath: "net/http"},
						"sync":     {Alias: "sync", PkgPath: "sync"},
					},
				}))
				g.Expect(f.Function).NotTo(BeZero())
				g.Expect(f.Packages).NotTo(BeNil())
				g.Expect(f.Package).NotTo(BeNil())
				g.Expect(f.Methods).To(HaveLen(0))
				g.Expect(f.Function.Name).To(Equal("HandlerFunc"))
				g.Expect(f.Function.Params).To(HaveLen(2))
				g.Expect(f.Function.Returns).To(BeEmpty())
			})
		})
	})

	when("working out the import path of a directory", func() {
		var fixtures string
		it.Before(func() {
			fixtures, err = filepath.Abs(filepath.Join("..", "fixtures"))
			g.Expect(err).NotTo(HaveOccurred())
		})

		it("joins the module path and the directory's path within the module", func() {
			path, ok := importPathForDir(filepath.Join(fixtures, "go-hyphenpackage"))
			g.Expect(ok).To(BeTrue())
			g.Expect(path).To(Equal("github.com/aidenwallis/counterfeiter/v7/fixtures/go-hyphenpackage"))
		})

		it("uses the module path for the module's root", func() {
			path, ok := importPathForDir(filepath.Dir(fixtures))
			g.Expect(ok).To(BeTrue())
			g.Expect(path).To(Equal("github.com/aidenwallis/counterfeiter/v7"))
		})

		it("uses the innermost module", func() {
			path, ok := importPathForDir(filepath.Join(fixtures, "dup_packages", "a"))
			g.Expect(ok).To(BeTrue())
			g.Expect(path).To(Equal("github.com/aidenwallis/counterfeiter/v7/fixtures/dup_packages/a"))
		})

		it("gives up on a directory the go command does not import by its path in the module", func() {
			for _, dir := range []string{"vendor", "testdata", "_hidden", ".hidden"} {
				_, ok := importPathForDir(filepath.Join(fixtures, dir, "pkg"))
				g.Expect(ok).To(BeFalse(), dir)
			}
		})

		it("gives up outside a module", func() {
			_, ok := importPathForDir(t.TempDir())
			g.Expect(ok).To(BeFalse())
		})
	})

	when("preloading targets with Preload()", func() {
		ids := func(pkgs []*packages.Package) []string {
			var result []string
			for _, p := range pkgs {
				result = append(result, p.ID)
			}
			return result
		}

		it("caches, for every target, the packages that loading it on its own would", func() {
			fixtures, err := filepath.Abs(filepath.Join("..", "fixtures"))
			g.Expect(err).NotTo(HaveOccurred())
			targets := []string{
				fixtures,
				filepath.Join(fixtures, "externaltest"),
				"io",
				"github.com/aidenwallis/counterfeiter/v7/fixtures/genericinterface",
				"github.com/aidenwallis/counterfeiter/v7/fixtures/samepackage",
			}
			c := &Cache{}
			g.Expect(Preload(c, "", targets...)).To(Succeed())
			for _, target := range targets {
				alone := &Fake{TargetPackage: target, Mode: Package}
				g.Expect(alone.loadPackages(&FakeCache{}, "")).To(Succeed())
				preloaded, ok := c.Load(target)
				g.Expect(ok).To(BeTrue(), target)
				g.Expect(ids(preloaded)).To(Equal(ids(alone.Packages)), target)
			}
		})

		it("generates the same fake from preloaded packages as from a fresh load", func() {
			c := &Cache{}
			g.Expect(Preload(c, "", "os", "io")).To(Succeed())
			for _, target := range []string{"os", "io"} {
				name := map[string]string{"os": "FileInfo", "io": "WriteCloser"}[target]
				preloaded, err := NewFake(InterfaceOrFunction, name, target, "Fake"+name, target+"fakes", "", "", c)
				g.Expect(err).NotTo(HaveOccurred())
				fresh, err := NewFake(InterfaceOrFunction, name, target, "Fake"+name, target+"fakes", "", "", &FakeCache{})
				g.Expect(err).NotTo(HaveOccurred())
				want, err := fresh.Generate(true)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(preloaded.Generate(true)).To(Equal(want))
			}
		})

		it("leaves a target that did not load cleanly to be loaded, and reported, on its own", func() {
			c := &Cache{}
			g.Expect(Preload(c, "", "os", "nonexistentpackage")).To(Succeed())
			_, ok := c.Load("os")
			g.Expect(ok).To(BeTrue())
			_, ok = c.Load("nonexistentpackage")
			g.Expect(ok).To(BeFalse())
			_, err := NewFake(InterfaceOrFunction, "NonExistent", "nonexistentpackage", "FakeNonExistent", "nonexistentpackagefakes", "", "", c)
			g.Expect(err).To(HaveOccurred())
		})
	})

	when("manually constructing a fake", func() {
		it.Before(func() {
			f = &Fake{Imports: newImports()}
		})

		when("duplicate import package names are added", func() {
			it.Before(func() {
				f.Imports.Add("sync", "sync")
				f.Imports.Add("sync", "github.com/aidenwallis/counterfeiter/v7/fixtures/sync")
				f.Imports.Add("sync", "github.com/aidenwallis/counterfeiter/v7/fixtures/othersync")
			})

			it("all packages have unique aliases", func() {
				g.Expect(f.Imports).To(BeEquivalentTo(Imports{
					ByAlias: map[string]Import{
						"sync":  {Alias: "sync", PkgPath: "sync"},
						"synca": {Alias: "synca", PkgPath: "github.com/aidenwallis/counterfeiter/v7/fixtures/sync"},
						"syncb": {Alias: "syncb", PkgPath: "github.com/aidenwallis/counterfeiter/v7/fixtures/othersync"},
					},
					ByPkgPath: map[string]Import{
						"sync": {Alias: "sync", PkgPath: "sync"},
						"github.com/aidenwallis/counterfeiter/v7/fixtures/sync":      {Alias: "synca", PkgPath: "github.com/aidenwallis/counterfeiter/v7/fixtures/sync"},
						"github.com/aidenwallis/counterfeiter/v7/fixtures/othersync": {Alias: "syncb", PkgPath: "github.com/aidenwallis/counterfeiter/v7/fixtures/othersync"},
					},
				}))
			})
		})

		when("inspecting the target", func() {
			when("the target is not set", func() {
				it("IsInterface() is false", func() {
					g.Expect(f.IsInterface()).To(BeFalse())
				})

				it("IsFunction() is false", func() {
					g.Expect(f.IsFunction()).To(BeFalse())
				})
			})

			it("recognises an interface", func() {
				f.Mode = InterfaceOrFunction
				f.TargetPackage = "os"
				f.TargetName = "FileInfo"
				g.Expect(f.loadPackages(&Cache{}, "")).To(Succeed())
				g.Expect(f.findPackage()).To(Succeed())
				g.Expect(f.IsInterface()).To(BeTrue())
				g.Expect(f.IsFunction()).To(BeFalse())
			})

			it("recognises a function", func() {
				f.Mode = InterfaceOrFunction
				f.TargetPackage = "net/http"
				f.TargetName = "HandlerFunc"
				g.Expect(f.loadPackages(&Cache{}, "")).To(Succeed())
				g.Expect(f.findPackage()).To(Succeed())
				g.Expect(f.IsInterface()).To(BeFalse())
				g.Expect(f.IsFunction()).To(BeTrue())
			})

			it("treats a struct as neither", func() {
				f.Mode = InterfaceOrFunction
				f.TargetPackage = "net/http"
				f.TargetName = "Client"
				g.Expect(f.loadPackages(&Cache{}, "")).To(Succeed())
				g.Expect(f.findPackage()).NotTo(Succeed())
				g.Expect(f.IsInterface()).To(BeFalse())
				g.Expect(f.IsFunction()).To(BeFalse())
			})
		})

		when("in interface mode", func() {
			it.Before(func() {
				f.Mode = InterfaceOrFunction
			})

			when("targeting the os.FileInfo interface", func() {
				it.Before(func() {
					f.TargetPackage = "os"
					f.TargetName = "FileInfo"
					c := &Cache{}
					err := f.loadPackages(c, "")
					g.Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		when("in package mode", func() {
			it.Before(func() {
				f.Mode = Package
			})

			when("targeting a nonexistent package", func() {
				it("returns an error", func() {
					f.TargetPackage = "counterfeiternonexistentpackage"
					c := &Cache{}
					err := f.loadPackages(c, "")
					g.Expect(err).To(HaveOccurred())
				})
			})

			when("targeting the os package", func() {
				it("loads the package, finds it even behind an invalid entry, and loads its methods", func() {
					f.TargetPackage = "os"
					g.Expect(f.loadPackages(&Cache{}, "")).To(Succeed())
					g.Expect(len(f.Packages)).To(BeNumerically(">=", 1))
					g.Expect(f.Packages[0].Name).To(Equal("os"))

					g.Expect(f.findPackage()).To(Succeed())
					g.Expect(f.Package).To(Equal(f.Packages[0]))

					f.Packages = append([]*packages.Package{{}}, f.Packages...)
					g.Expect(f.findPackage()).To(Succeed())
					g.Expect(f.Package).To(Equal(f.Packages[1]))

					methods := packageMethodSet(f.Package)
					g.Expect(len(methods)).To(BeNumerically(">=", 51)) // yes, this is crazy because go 1.11 added a function

					g.Expect(f.loadMethods()).To(Succeed())
					g.Expect(len(f.Methods)).To(BeNumerically(">=", 51)) // yes, this is crazy because go 1.11 added a function
					g.Expect(len(f.Imports.ByAlias)).To(Equal(3))
				})
			})
		})

		when("working with imports", func() {
			when("there are no imports", func() {
				it("returns an empty alias map", func() {
					g.Expect(f.Imports.ByAlias).To(BeEmpty())
				})

				it("turns a vendor path into the correct import", func() {
					i := f.Imports.Add("apackage", "github.com/aidenwallis/counterfeiter/v7/fixtures/vendored/vendor/apackage")
					g.Expect(i.Alias).To(Equal("apackage"))
					g.Expect(i.PkgPath).To(Equal("apackage"))

					i = f.Imports.Add("anotherpackage", "vendor/anotherpackage")
					g.Expect(i.Alias).To(Equal("anotherpackage"))
					g.Expect(i.PkgPath).To(Equal("anotherpackage"))
				})
			})

			when("there is a single import", func() {
				it.Before(func() {
					f.Imports.Add("os", "os")
				})

				it("is present in the map", func() {
					g.Expect(f.Imports).To(BeEquivalentTo(Imports{
						ByAlias: map[string]Import{
							"os": {Alias: "os", PkgPath: "os"},
						},
						ByPkgPath: map[string]Import{
							"os": {Alias: "os", PkgPath: "os"},
						},
					}))
				})

				it("returns the existing imports if there is a path match", func() {
					i := f.Imports.Add("aliasedos", "os")
					g.Expect(i.Alias).To(Equal("os"))
					g.Expect(i.PkgPath).To(Equal("os"))
					g.Expect(f.Imports).To(BeEquivalentTo(Imports{
						ByAlias: map[string]Import{
							"os": {Alias: "os", PkgPath: "os"},
						},
						ByPkgPath: map[string]Import{
							"os": {Alias: "os", PkgPath: "os"},
						},
					}))
				})
			})
		})
	})

	when("helper functions", func() {
		when("unexport()", func() {
			it("is a no-op on an empty string", func() {
				g.Expect(unexport("")).To(Equal(""))
				g.Expect(unexport(" ")).To(Equal(""))
			})

			it("makes the first letter lowercase", func() {
				g.Expect(unexport("TheExportedThing")).To(Equal("theExportedThing"))
			})

			it("leaves unexported things unchanged", func() {
				g.Expect(unexport("theUnexportedThing")).To(Equal("theUnexportedThing"))
			})
		})

		when("isBuildTranscript()", func() {
			it("recognises the compiler output go list -export attaches to a package that failed to build", func() {
				g.Expect(isBuildTranscript(packages.Error{Kind: packages.ListError, Msg: "# example.com/widgets\n./fake_widget.go:112:16: missing method Undo"})).To(BeTrue())
			})

			it("leaves positioned and non-build errors alone", func() {
				g.Expect(isBuildTranscript(packages.Error{Kind: packages.TypeError, Pos: "/a/fake_widget.go:112:16", Msg: "missing method Undo"})).To(BeFalse())
				g.Expect(isBuildTranscript(packages.Error{Kind: packages.ListError, Pos: "/a/widgets.go:5:2", Msg: "no required module provides package x.invalid/nope"})).To(BeFalse())
				g.Expect(isBuildTranscript(packages.Error{Kind: packages.ParseError, Msg: "# example.com/widgets"})).To(BeFalse())
			})
		})

		when("hasInvalidType()", func() {
			invalid := types.Typ[types.Invalid]
			str := types.Typ[types.String]
			named := func(underlying types.Type) *types.Named {
				return types.NewNamed(types.NewTypeName(0, nil, "Named", nil), underlying, nil)
			}

			it("is true for a type the loader could not resolve", func() {
				g.Expect(hasInvalidType(invalid)).To(BeTrue())
			})

			it("is false for valid types", func() {
				g.Expect(hasInvalidType(nil)).To(BeFalse())
				g.Expect(hasInvalidType(str)).To(BeFalse())
				g.Expect(hasInvalidType(types.NewPointer(named(str)))).To(BeFalse())
			})

			it("looks through composite types", func() {
				g.Expect(hasInvalidType(types.NewPointer(invalid))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewSlice(invalid))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewArray(invalid, 2))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewChan(types.SendRecv, invalid))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewMap(str, invalid))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewMap(invalid, str))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewStruct([]*types.Var{types.NewField(0, nil, "f", invalid, false)}, nil))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(0, nil, "p", invalid)), nil, false))).To(BeTrue())
				g.Expect(hasInvalidType(types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewParam(0, nil, "r", invalid)), false))).To(BeTrue())
			})

			it("does not look through a named type, which prints by name", func() {
				g.Expect(hasInvalidType(named(invalid))).To(BeFalse())
				g.Expect(hasInvalidType(types.NewPointer(named(invalid)))).To(BeFalse())
			})
		})

		when("isExported()", func() {
			it("returns false for an empty string", func() {
				g.Expect(isExported("")).To(BeFalse())
				g.Expect(isExported(" ")).To(BeFalse())
			})

			it("returns true when the first rune is upper case", func() {
				g.Expect(isExported("Identifier")).To(BeTrue())
				g.Expect(isExported("Ʊpsilon")).To(BeTrue())
			})

			it("returns false when the first rune not upper case", func() {
				g.Expect(isExported("identifier")).To(BeFalse())
				g.Expect(isExported("ʊpsilon")).To(BeFalse())
			})
		})
	})
}
