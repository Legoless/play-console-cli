package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/webdriver"
	"github.com/tamtom/play-console-cli/internal/websession"
)

// declarationsResult is the `web apps declarations` output.
type declarationsResult struct {
	PackageName string                    `json:"packageName"`
	Declarations *webdriver.DeclarationsState `json:"declarations,omitempty"`
	Changed     bool                      `json:"changed"`
	Set         string                    `json:"set,omitempty"`
	Value       string                    `json:"value,omitempty"`
}

// settableRadioDeclarations maps --set keys to console app-content routes.
var settableRadioDeclarations = map[string]string{
	"government-apps":     "government-apps",
	"ads-declaration":     "ads-declaration",
	"testing-credentials": "testing-credentials",
}

// WebAppsDeclarationsCommand returns the `gplay web apps declarations` subcommand.
func WebAppsDeclarationsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps declarations", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	set := fs.String("set", "", "Declaration to change: privacy-policy, government-apps, ads-declaration, testing-credentials")
	url := fs.String("url", "", "Privacy policy URL (with --set privacy-policy)")
	value := fs.String("value", "", "Answer for radio declarations: yes or no")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm the declaration change")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "declarations",
		ShortUsage: "gplay web apps declarations --package <id> [--set <key> [--url <url> | --value <yes|no>] --confirm]",
		ShortHelp:  "Read App content declarations, or set the simple ones.",
		LongHelp: `Read the App content policy declarations, or change the simple ones.

Without --set this lists every declaration with its status and last-edited
date, plus anything needing attention. Setting requires --confirm and
verifies the persisted answer afterwards.

Settable declarations:
  privacy-policy       --url <https://...>
  government-apps      --value yes|no
  ads-declaration      --value yes|no
  testing-credentials  --value no (adding credentials needs the console)

Content rating, target audience, data safety, health, and financial features
are multi-step questionnaires; answer them in the console. These commands
drive the console because the official Android Publisher API has no
declaration endpoints. Always confirm changes with the user first.

Examples:
  gplay web apps declarations --package com.example.app
  gplay web apps declarations --package com.example.app --set privacy-policy --url https://example.com/privacy --confirm
  gplay web apps declarations --package com.example.app --set government-apps --value no --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			setKey := strings.TrimSpace(*set)
			write := setKey != ""
			if write {
				if setKey == "privacy-policy" {
					if strings.TrimSpace(*url) == "" {
						return fmt.Errorf("--url is required with --set privacy-policy")
					}
				} else if _, ok := settableRadioDeclarations[setKey]; ok {
					v := strings.ToLower(strings.TrimSpace(*value))
					if v != "yes" && v != "no" {
						return fmt.Errorf("--value must be yes or no")
					}
					if setKey == "testing-credentials" && v == "yes" {
						return fmt.Errorf("restricted sign in details must be added in the console; only 'no' can be set here")
					}
				} else {
					return fmt.Errorf("--set must be one of: privacy-policy, government-apps, ads-declaration, testing-credentials")
				}
				if !*confirm {
					return fmt.Errorf("--confirm is required to change a declaration")
				}
			}

			if shared.IsDryRun(ctx) {
				if write {
					fmt.Fprintf(os.Stderr, "[DRY RUN] would set declaration %s: package=%s\n", setKey, packageName) // #nosec G705 -- stderr log
				} else {
					fmt.Fprintf(os.Stderr, "[DRY RUN] would read declarations: package=%s\n", packageName) // #nosec G705 -- stderr log
				}
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}
			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer pb.Close()

			result := &declarationsResult{PackageName: packageName}
			if !write {
				state, err := pb.ReadDeclarations(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
				if err != nil {
					return err
				}
				result.Declarations = state
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}

			result.Set = setKey
			if setKey == "privacy-policy" {
				u := strings.TrimSpace(*url)
				changed, err := pb.SetPrivacyPolicyURL(ctx, target.DeveloperID, target.AppID, sess.UserEmail, u)
				if err != nil {
					return err
				}
				result.Value = u
				result.Changed = changed
			} else {
				yes := strings.EqualFold(strings.TrimSpace(*value), "yes")
				changed, err := pb.SetRadioDeclaration(ctx, target.DeveloperID, target.AppID, sess.UserEmail, settableRadioDeclarations[setKey], yes)
				if err != nil {
					return err
				}
				result.Value = strings.ToLower(strings.TrimSpace(*value))
				result.Changed = changed
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// WebAppsPolicyCommand returns the `gplay web apps policy` subcommand.
func WebAppsPolicyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps policy", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "policy",
		ShortUsage: "gplay web apps policy --package <id>",
		ShortHelp:  "Read the app's Play policy status and reported issues.",
		LongHelp: `Read the Policy status page: reported policy issues, or the empty
state shown when nothing is reported. Before the first review completes, the
page shows that compliance information appears after review. Read-only.

Examples:
  gplay web apps policy --package com.example.app
  gplay web apps policy --package com.example.app --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}
			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer pb.Close()
			status, err := pb.ReadPolicyStatus(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			return shared.PrintOutput(status, *outputFlag, *pretty)
		},
	}
}

// publishResult is the `web apps publish` output.
type publishResult struct {
	PackageName       string `json:"packageName"`
	ManagedPublishing bool   `json:"managedPublishing"`
	Published         bool   `json:"published"`
	Changed           bool   `json:"changed"`
}

// WebAppsPublishCommand returns the `gplay web apps publish` subcommand.
func WebAppsPublishCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps publish", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	managed := fs.String("managed", "", "Set managed publishing: on or off")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm publishing / the managed publishing change")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "publish",
		ShortUsage: "gplay web apps publish --package <id> [--managed on|off --confirm | --confirm]",
		ShortHelp:  "Publish approved changes, or toggle managed publishing.",
		LongHelp: `Publish approved changes from the Publishing overview, or toggle
managed publishing.

Without flags this reports the managed publishing state and whether anything
is publishable. With --managed on|off (and --confirm) it toggles managed
publishing and verifies the new state. With --confirm alone it clicks the
Publish action that appears once Google approves the reviewed changes.

These drive the Publishing overview because the official Android Publisher
API has no publish endpoint. Always confirm with the user first.

Examples:
  gplay web apps publish --package com.example.app
  gplay web apps publish --package com.example.app --managed on --confirm
  gplay web apps publish --package com.example.app --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			managedFlag := strings.ToLower(strings.TrimSpace(*managed))
			if managedFlag != "" && managedFlag != "on" && managedFlag != "off" {
				return fmt.Errorf("--managed must be on or off")
			}
			if managedFlag != "" && !*confirm {
				return fmt.Errorf("--confirm is required to change managed publishing")
			}

			if shared.IsDryRun(ctx) {
				fmt.Fprintf(os.Stderr, "[DRY RUN] would publish/manage: package=%s managed=%q confirm=%v\n", packageName, managedFlag, *confirm) // #nosec G705 -- stderr log
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}
			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer pb.Close()

			result := &publishResult{PackageName: packageName}
			if managedFlag != "" {
				if err := pb.SetManagedPublishing(ctx, target.DeveloperID, target.AppID, sess.UserEmail, managedFlag == "on"); err != nil {
					return err
				}
				result.Changed = true
			}
			overview, err := pb.ReadOverview(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			result.ManagedPublishing = overview.ManagedPublishing
			if managedFlag != "" {
				// Toggle-only run: report, no publish attempt.
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}
			if !*confirm {
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}
			if err := pb.PublishApprovedChanges(ctx, target.DeveloperID, target.AppID, sess.UserEmail, 2*time.Minute); err != nil {
				return err
			}
			result.Published = true
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
