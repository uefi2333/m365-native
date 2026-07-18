package auth

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultAuthorityIsSingleTenant(t *testing.T) {
	t.Setenv("M365_AUTHORITY", "")
	want := "https://login.microsoftonline.com/" + DefaultTenantID
	if got := Authority(); got != want {
		t.Fatalf("Authority() = %q, want %q", got, want)
	}
	if strings.Contains(AuthorizeEndpoint(), "/common/") {
		t.Fatal("default authorize endpoint must not be multitenant")
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
