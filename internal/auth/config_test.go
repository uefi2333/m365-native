package auth

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultAuthorityIsMultitenant(t *testing.T) {
	t.Setenv("M365_AUTHORITY", "")
	if got := Authority(); got != "https://login.microsoftonline.com/common" {
		t.Fatalf("Authority() = %q", got)
	}
	if strings.Contains(AuthorizeEndpoint(), "f7c4604c-0ec5-4d52-90eb-68db37632328") {
		t.Fatal("default authorize endpoint still uses the invalid tenant")
	}
}

func TestAuthorityOverride(t *testing.T) {
	const custom = "https://login.microsoftonline.com/organizations"
	if err := os.Setenv("M365_AUTHORITY", custom); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("M365_AUTHORITY")
	if got := Authority(); got != custom {
		t.Fatalf("Authority() = %q", got)
	}
}
