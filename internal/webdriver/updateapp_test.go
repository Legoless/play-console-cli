package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAppSettingsURL(t *testing.T) {
	got := appSettingsURL("6901885972034847549", "4974539508825146246", "dal@unifiedsense.com")
	want := "https://play.google.com/console/developers/6901885972034847549/app/4974539508825146246/store-settings?authuser=dal%40unifiedsense.com&hl=en"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestReadAppSettings(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(readAppSettingsScript, map[string]any{"kind": "game", "category": "Educational", "canSubmit": true})
	b, err := Connect(context.Background(), portDir(t, f), 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Close()

	state, err := ReadAppSettings(context.Background(), b)
	if err != nil {
		t.Fatalf("ReadAppSettings: %v", err)
	}
	if state.Kind != "game" || state.Category != "Educational" || !state.CanSubmit {
		t.Errorf("state = %+v", state)
	}
}

func TestSetAppClassification_RejectsMissingControl(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(setDropdownScript("type-dropdown", "game"), "application type control not found")
	b, err := Connect(context.Background(), portDir(t, f), 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Close()

	err = SetAppClassification(context.Background(), b, "game", "Educational")
	if err == nil || !strings.Contains(err.Error(), "control not found") {
		t.Errorf("err = %v", err)
	}
}

func TestSubmitAppSettings_ErrorsWhenSaveDisabled(t *testing.T) {
	f := newFakeChrome(t)
	b, err := Connect(context.Background(), portDir(t, f), 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Close()

	err = SubmitAppSettings(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "Save") {
		t.Errorf("err = %v, want disabled Save error", err)
	}
}

func TestSubmitAppSettings_AcceptsDisabledSaveWithDialogOpen(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(submitAppSettingsScript, true)
	f.setReply(appSettingsSaveSettledScript, true)
	b, err := Connect(context.Background(), portDir(t, f), 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Close()

	if err := SubmitAppSettings(context.Background(), b, 300*time.Millisecond); err != nil {
		t.Fatalf("SubmitAppSettings: %v", err)
	}
}
