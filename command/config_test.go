package command_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aidenwallis/counterfeiter/v6/command"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestConfig(t *testing.T) {
	spec.Run(t, "Config", testConfig, spec.Report(report.Terminal{}))
}

func testConfig(t *testing.T, when spec.G, it spec.S) {
	var root string

	write := func(files map[string]string) {
		for name, content := range files {
			p := filepath.Join(root, name)
			Expect(os.MkdirAll(filepath.Dir(p), 0o777)).To(Succeed())
			Expect(os.WriteFile(p, []byte(content), 0o666)).To(Succeed())
		}
	}
	detect := func(dir string) ([]command.Invocation, error) {
		return command.Detect(filepath.Join(root, dir), []string{"counterfeiter"}, false)
	}

	it.Before(func() {
		RegisterTestingT(t)
		var err error
		root, err = os.MkdirTemp("", "counterfeiter-config")
		Expect(err).NotTo(HaveOccurred())
		root, err = filepath.EvalSymlinks(root)
		Expect(err).NotTo(HaveOccurred())
		write(map[string]string{
			"go.mod":             "module example.com/app\n\ngo 1.26\n",
			"store/store.go":     "package store\n\ntype Repository interface{ Get() }\n\ntype Presenter interface{ Show() }\n",
			"jobs/jobs.go":       "package jobs\n\n//counterfeiter:generate . Queue\ntype Queue interface{ Push() }\n",
			"jobs/jobs_test.go":  "package jobs\n\n//counterfeiter:generate -o fake . Worker\ntype Worker interface{ Work() }\n",
			"jobs/cron/cron.go":  "package cron\n\n//counterfeiter:generate . Schedule\ntype Schedule interface{ Next() }\n",
			"other/other.go":     "package other\n\n//counterfeiter:generate . Ignored\ntype Ignored interface{ No() }\n",
			"hack/header.txt":    "// header\n",
			"store/sub/empty.go": "package sub\n",
		})
	})

	it.After(func() {
		Expect(os.RemoveAll(root)).To(Succeed())
	})

	when("there is no config file", func() {
		it("leaves the bare invocation alone, so it is reported as a usage error", func() {
			i, err := detect("store")
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(1))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter"}))
		})

		it("does not look above the module root", func() {
			outer := root
			root = filepath.Join(root, "nested")
			write(map[string]string{"go.mod": "module example.com/nested\n\ngo 1.26\n"})
			write(map[string]string{"../.counterfeiter.yaml": "packages: {}\n"})
			i, err := detect("")
			root = outer
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(1))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter"}))
		})
	})

	when("a package lists its interfaces", func() {
		it("fakes each one as if a directive in that package named it, whether the package is given by path or import path", func() {
			write(map[string]string{".counterfeiter.yaml": `
packages:
  ./store:
    interfaces:
      Repository:
  example.com/app/jobs:
    interfaces:
      Queue: {}
`})
			i, err := detect("")
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(2))
			Expect(i[0].Dir).To(Equal(filepath.Join(root, "store")))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter", ".", "Repository"}))
			Expect(i[0].File).To(Equal(filepath.Join(root, ".counterfeiter.yaml")))
			Expect(i[1].Dir).To(Equal(filepath.Join(root, "jobs")))
			Expect(i[1].Args).To(Equal([]string{"counterfeiter", ".", "Queue"}))
		})

		it("is found from a directory below the config file", func() {
			write(map[string]string{".counterfeiter.yaml": "packages:\n  ./store:\n    interfaces:\n      Repository:\n"})
			i, err := detect(filepath.Join("store", "sub"))
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(1))
			Expect(i[0].Dir).To(Equal(filepath.Join(root, "store")))
		})

		it("lets an interface's options override its package's, which override the top level's", func() {
			write(map[string]string{".counterfeiter.yaml": `
o: fakes
q: true
fake-name-template: "{{.TargetName}}Double"
packages:
  ./store:
    test: true
    o: storetest
    interfaces:
      Presenter:
      Repository:
        fake-name: Repo
        o: repo
        test: false
`})
			i, err := detect("")
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(2))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter", "-o", "storetest", "-fake-name-template", "{{.TargetName}}Double", "-test", "-q", ".", "Presenter"}))
			Expect(i[1].Args).To(Equal([]string{"counterfeiter", "-o", "repo", "-fake-name", "Repo", "-fake-name-template", "{{.TargetName}}Double", "-q", ".", "Repository"}))
		})

		it("reads the header relative to the config file", func() {
			write(map[string]string{".counterfeiter.yaml": "header: hack/header.txt\npackages:\n  ./store:\n    interfaces:\n      Repository:\n"})
			i, err := detect("")
			Expect(err).NotTo(HaveOccurred())
			Expect(i[0].Args).To(Equal([]string{"counterfeiter", "-header", filepath.Join(root, "hack", "header.txt"), ".", "Repository"}))
		})

		it("fakes a package outside the module from the config file's directory", func() {
			write(map[string]string{".counterfeiter.yaml": "packages:\n  io:\n    o: internal/iofakes\n    interfaces:\n      WriteCloser:\n"})
			i, err := detect("")
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(1))
			Expect(i[0].Dir).To(Equal(root))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter", "-o", "internal/iofakes", "io.WriteCloser"}))
		})

		it("requires o for a package outside the module, which has no directory of its own to write into", func() {
			write(map[string]string{".counterfeiter.yaml": "packages:\n  io:\n    interfaces:\n      WriteCloser:\n"})
			_, err := detect("")
			Expect(err).To(MatchError(ContainSubstring("io")))
			Expect(err).To(MatchError(ContainSubstring("o")))
		})

		it("rejects a package that lists no interfaces", func() {
			write(map[string]string{".counterfeiter.yaml": "packages:\n  ./store: {}\n"})
			_, err := detect("")
			Expect(err).To(MatchError(ContainSubstring("./store")))
		})

		it("rejects keys it does not know", func() {
			write(map[string]string{".counterfeiter.yaml": "packages:\n  ./store:\n    interface:\n      Repository:\n"})
			_, err := detect("")
			Expect(err).To(MatchError(ContainSubstring("interface")))
		})
	})

	when("directives names packages", func() {
		it("runs the //counterfeiter:generate directives of every matching package, with the top level's options as defaults", func() {
			write(map[string]string{".counterfeiter.yaml": "header: hack/header.txt\nq: true\ndirectives:\n  - ./jobs/...\n"})
			i, err := detect("")
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(HaveLen(3))
			header := filepath.Join(root, "hack", "header.txt")
			Expect(i[0].Dir).To(Equal(filepath.Join(root, "jobs")))
			Expect(i[0].File).To(Equal("jobs.go"))
			Expect(i[0].Line).To(Equal(3))
			Expect(i[0].Args).To(Equal([]string{"counterfeiter", "-header", header, "-q", ".", "Queue"}))
			Expect(i[1].Dir).To(Equal(filepath.Join(root, "jobs")))
			Expect(i[1].Args).To(Equal([]string{"counterfeiter", "-header", header, "-q", "-o", "fake", ".", "Worker"}))
			Expect(i[2].Dir).To(Equal(filepath.Join(root, "jobs", "cron")))
			Expect(i[2].Args).To(Equal([]string{"counterfeiter", "-header", header, "-q", ".", "Schedule"}))
		})
	})
}
