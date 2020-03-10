package layer_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestWindowsWriter(t *testing.T) {
	spec.Run(t, "windows-writer", testWindowsWriter, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testWindowsWriter(t *testing.T, when spec.G, it spec.S) {
	when("#Write", func() {
		it("does", func() {
		})
	})

	when("#WriteHeader", func() {
		it("does", func() {
		})
	})
}
