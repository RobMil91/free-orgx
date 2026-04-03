package setup_test

import (
	"os"
	"testing"

	"github.com/RobMil91/free-orgx/config"
	"github.com/RobMil91/free-orgx/internal/setup"
)

func TestSetup_RAMDB(t *testing.T) {
	cfg := config.Config{
		RAMDB: true,
	}

	adapters, err := setup.Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() failed: %v", err)
	}

	if adapters.UserRep == nil {
		t.Fatal("UserRep should not be nil")
	}
	if adapters.ProjectRep == nil {
		t.Fatal("ProjectRep should not be nil")
	}
}

func TestSetup_SQLite(t *testing.T) {
	t.Setenv("PASSWORD_PEPPER", "test-pepper")

	cfg := config.Config{
		RAMDB: false,
	}

	adapters, err := setup.Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() failed: %v", err)
	}

	if adapters.UserRep == nil {
		t.Fatal("UserRep should not be nil")
	}
	if adapters.ProjectRep == nil {
		t.Fatal("ProjectRep should not be nil")
	}
}

func TestSetup_SQLite_NoPepper(t *testing.T) {
	os.Unsetenv("PASSWORD_PEPPER")

	cfg := config.Config{
		RAMDB: false,
	}

	adapters, err := setup.Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() should not fail without pepper: %v", err)
	}

	if adapters.UserRep == nil {
		t.Fatal("UserRep should not be nil even without pepper")
	}
}
