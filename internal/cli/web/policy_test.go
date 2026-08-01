package web

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/webdriver"
)

// --- declarations ---

func TestWebAppsDeclarations_ValidatesFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: []string{"--set", "government-apps", "--value", "no", "--confirm"}, want: "--package"},
		{name: "url required", args: availabilityArgs("--set", "privacy-policy", "--confirm"), want: "--url"},
		{name: "value required", args: availabilityArgs("--set", "government-apps", "--confirm"), want: "--value must be yes or no"},
		{name: "unknown set", args: availabilityArgs("--set", "data-safety", "--value", "no", "--confirm"), want: "--set must be one of"},
		{name: "credentials yes", args: availabilityArgs("--set", "testing-credentials", "--value", "yes", "--confirm"), want: "console"},
		{name: "confirm", args: availabilityArgs("--set", "government-apps", "--value", "no"), want: "--confirm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsDeclarationsCommand()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestWebAppsDeclarations_ReadAll(t *testing.T) {
	f := &fakePublishBrowser{
		declarations: &webdriver.DeclarationsState{
			Actioned: []webdriver.Declaration{
				{Key: "privacy-policy", Title: "Privacy policy", Status: "Completed", LastEdited: "Jul 29, 2026"},
				{Key: "government-apps", Title: "Government apps", Status: "Completed"},
			},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsDeclarationsCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("declarations: %v", err)
	}
	for _, want := range []string{`"key":"privacy-policy"`, `"key":"government-apps"`, "Jul 29, 2026", `"changed":false`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %s", want, stdout)
		}
	}
}

func TestWebAppsDeclarations_DryRunSkipsBrowser(t *testing.T) {
	f := &fakePublishBrowser{failAt: "set-radio"}
	stubPublishBrowser(t, f)

	cmd := WebAppsDeclarationsCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--set", "government-apps", "--value", "no", "--confirm")); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(f.steps) != 0 {
		t.Errorf("browser steps = %v, want none", f.steps)
	}
}

func TestWebAppsDeclarations_SetsRadioDeclaration(t *testing.T) {
	f := &fakePublishBrowser{}
	setupPublish(t, f)

	cmd := WebAppsDeclarationsCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--set", "government-apps", "--value", "no", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("declarations set: %v", err)
	}
	if f.setRadioPage != "government-apps" || f.setRadioYes {
		t.Errorf("set radio = %s/%v, want government-apps/no", f.setRadioPage, f.setRadioYes)
	}
	for _, want := range []string{`"set":"government-apps"`, `"value":"no"`, `"changed":true`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %s", want, stdout)
		}
	}
}

func TestWebAppsDeclarations_SetsPrivacyPolicyURL(t *testing.T) {
	f := &fakePublishBrowser{}
	setupPublish(t, f)

	cmd := WebAppsDeclarationsCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--set", "privacy-policy", "--url", "https://example.com/privacy", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("declarations set: %v", err)
	}
	if f.setPrivacyURL != "https://example.com/privacy" {
		t.Errorf("set url = %q", f.setPrivacyURL)
	}
	if !strings.Contains(stdout, `"changed":true`) {
		t.Errorf("output = %s, want changed", stdout)
	}
}

// --- policy ---

func TestWebAppsPolicy_ValidatesPackage(t *testing.T) {
	cmd := WebAppsPolicyCommand()
	if err := cmd.FlagSet.Parse(nil); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--package") {
		t.Errorf("err = %v, want --package error", err)
	}
}

func TestWebAppsPolicy_ReadsStatus(t *testing.T) {
	f := &fakePublishBrowser{
		policy: &webdriver.PolicyStatus{State: "empty", Message: "Information about your compliance will be shown here after review"},
	}
	setupPublish(t, f)

	cmd := WebAppsPolicyCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if !strings.Contains(stdout, `"state":"empty"`) || !strings.Contains(stdout, "compliance") {
		t.Errorf("output = %s, want empty state with message", stdout)
	}
}

// --- publish ---

func TestWebAppsPublish_ValidatesFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: []string{"--managed", "on", "--confirm"}, want: "--package"},
		{name: "managed value", args: availabilityArgs("--managed", "maybe", "--confirm"), want: "--managed must be on or off"},
		{name: "confirm", args: availabilityArgs("--managed", "on"), want: "--confirm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsPublishCommand()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestWebAppsPublish_ReadsState(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{{ManagedPublishing: false}}}
	setupPublish(t, f)

	cmd := WebAppsPublishCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("publish read: %v", err)
	}
	if !strings.Contains(stdout, `"managedPublishing":false`) || !strings.Contains(stdout, `"published":false`) {
		t.Errorf("output = %s", stdout)
	}
	if slices.Contains(f.steps, "publish-now") {
		t.Error("must not publish without --confirm")
	}
}

func TestWebAppsPublish_TogglesManagedPublishing(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{{ManagedPublishing: true}}}
	setupPublish(t, f)

	cmd := WebAppsPublishCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--managed", "on", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("publish toggle: %v", err)
	}
	if !f.setManagedOn {
		t.Error("setManagedOn = false, want true")
	}
	if !strings.Contains(stdout, `"changed":true`) || !strings.Contains(stdout, `"managedPublishing":true`) {
		t.Errorf("output = %s", stdout)
	}
	if slices.Contains(f.steps, "publish-now") {
		t.Error("managed toggle must not publish")
	}
}

func TestWebAppsPublish_PublishesApprovedChanges(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{{}}}
	setupPublish(t, f)

	cmd := WebAppsPublishCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !slices.Contains(f.steps, "publish-now") {
		t.Errorf("steps = %v, want publish-now", f.steps)
	}
	if !strings.Contains(stdout, `"published":true`) {
		t.Errorf("output = %s, want published", stdout)
	}
}
