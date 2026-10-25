package seeded_test

import (
	"testing"

	"github.com/aidenwallis/counterfeiter/v7/fixtures/seeded"
	"github.com/aidenwallis/counterfeiter/v7/fixtures/seeded/fakes"
)

func TestFakeJoinsTheSeededPackage(t *testing.T) {
	var s seeded.Sower = &seeded_fakes.FakeSower{}
	if _, err := s.Sow("x"); err != nil {
		t.Fatalf("Sow returned error: %v", err)
	}
}
