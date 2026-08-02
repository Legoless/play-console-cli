package webdriver

import (
	"strings"
	"testing"
)

func scriptSection(t *testing.T, script, start, end string) string {
	t.Helper()
	_, section, ok := strings.Cut(script, start)
	if !ok {
		t.Fatalf("script is missing %q", start)
	}
	section, _, ok = strings.Cut(section, end)
	if !ok {
		t.Fatalf("script section %q is missing terminator %q", start, end)
	}
	return section
}

func TestDistributionHelpers_FallsBackToManageRowText(t *testing.T) {
	factors := scriptSection(t, distributionHelpers, "const factors", "const tasks")
	readsAncestor := strings.Contains(factors, ".closest(") || strings.Contains(factors, ".parentElement")
	readsText := strings.Contains(factors, ".textContent") || strings.Contains(factors, ".innerText")
	if !readsAncestor || !readsText {
		t.Error("factor lookup must fall back to the manage button row text when aria-label is absent")
	}
}

func TestDistributionHelpers_SelectsNamedVisibleAddButton(t *testing.T) {
	add := scriptSection(t, distributionHelpers, "const addButton", "const menuItem")
	if !strings.Contains(add, "querySelectorAll('[debug-id=text-button]')") ||
		!strings.Contains(add, "Add form factor") ||
		!strings.Contains(add, "getClientRects().length") {
		t.Error("add button lookup must select the visible Add form factor control")
	}
}

func TestDistributionHelpers_DoesNotAcceptLegalTerms(t *testing.T) {
	if strings.Contains(distributionHelpers, "accept-button") {
		t.Error("adding a form factor must leave legal opt-in terms for explicit follow-up")
	}
}
