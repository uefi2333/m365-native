package web

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapPasswordUsesWritablePersistentPath(t *testing.T) {
	dir := t.TempDir()
	persisted := filepath.Join(dir, "data", "admin-password")
	bootstrap := filepath.Join(dir, "secret")
	if err := os.WriteFile(bootstrap, []byte("bootstrap-password\n"), 0400); err != nil {
		t.Fatal(err)
	}
	t.Setenv("M365_ADMIN_PASSWORD_FILE", persisted)
	t.Setenv("M365_ADMIN_PASSWORD_BOOTSTRAP_FILE", bootstrap)
	t.Setenv("M365_ADMIN_PASSWORD", "")

	got, mustChange := loadAdminPassword()
	if got != "bootstrap-password" || mustChange {
		t.Fatalf("loadAdminPassword()=(%q,%v)", got, mustChange)
	}
	if err := saveAdminPassword("a-new-password-123"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "a-new-password-123\n" {
		t.Fatalf("persisted password=%q", b)
	}
}
