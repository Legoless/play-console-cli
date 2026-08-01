package webdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func productionTrackURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "tracks/production")
}

// rolloutDraftTimeout bounds the wait for a draft release to appear. It is a
// var so tests can shorten it.
var rolloutDraftTimeout = 30 * time.Second

// rolloutHelpers installs release-wizard lookups. Rolling out a draft release
// is a wizard: Releases tab on the track page, "Edit release" (prepare page),
// "Next" (review page), "Save" — which stages the release in the Publishing
// overview. The overview's own send-for-review then submits it.
const rolloutHelpers = `
(() => {
  const btn = sel => {
    const w = document.querySelector(sel);
    return w && (w.querySelector('button') || w);
  };
  const editDraftButton = () => btn('[debug-id=edit-draft-release-button]');
  const nextButton = () => btn('[debug-id=review-button]');
  const saveButton = () => btn('[debug-id=publishing-bottom-bar] [debug-id=main-button]');
  const releaseName = () => {
    const i = document.querySelector('[debug-id=version] input');
    return i ? (i.value || '').trim() : '';
  };
  const warnings = () => [...document.querySelectorAll('[debug-id=validation-description]')]
    .map(e => (e.innerText || '').replace(/\s+/g, ' ').trim())
    .filter(Boolean);
  const enabled = b => !!(b && !b.disabled && b.getAttribute('aria-disabled') !== 'true');
  window.__gplayRollout = { editDraftButton, nextButton, saveButton, releaseName, warnings, enabled };
  return true;
})()
`

const openReleasesTabScript = `(() => {
  const tab = [...document.querySelectorAll('[role=tab]')].find(t => /^releases$/i.test(t.textContent.trim()));
  if (!tab) return false;
  tab.click();
  return true;
})()`

// OpenProductionReleases opens the production track's Releases tab.
func OpenProductionReleases(ctx context.Context, b *Browser, developerID, appID, account string) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, productionTrackURL(developerID, appID, account)); err != nil {
		return err
	}
	tabReady := formHelpers + ` && ` + rolloutHelpers +
		` && !! [...document.querySelectorAll('[role=tab]')].find(t => /^releases$/i.test(t.textContent.trim()))`
	if err := b.EvalUntil(ctx, tabReady, 60*time.Second); err != nil {
		return fmt.Errorf("production track page did not load (is the gplay browser profile signed in?): %w", err)
	}
	var opened bool
	if err := b.Eval(ctx, openReleasesTabScript, &opened); err != nil {
		return err
	}
	if !opened {
		return fmt.Errorf("Releases tab was not found on the production track page")
	}
	draftReady := formHelpers + ` && ` + rolloutHelpers + ` && !!window.__gplayRollout.editDraftButton()`
	if err := b.EvalUntil(ctx, draftReady, rolloutDraftTimeout); err != nil {
		return fmt.Errorf("no draft release found on the production track: %w", err)
	}
	return nil
}

const editDraftClickScript = `(() => {
  const r = window.__gplayRollout;
  const btn = r && r.editDraftButton();
  if (!r.enabled(btn)) return false;
  btn.click();
  return true;
})()`

// PrepareState is the release prepare page state.
type PrepareState struct {
	ReleaseName string `json:"releaseName"`
}

const readPrepareScript = `(() => ({
  releaseName: window.__gplayRollout.releaseName(),
}))()`

// OpenDraftRelease opens the draft release's prepare page and reads it.
func OpenDraftRelease(ctx context.Context, b *Browser) (*PrepareState, error) {
	var clicked bool
	if err := b.Eval(ctx, formHelpers+` && `+rolloutHelpers+` && `+editDraftClickScript, &clicked); err != nil {
		return nil, err
	}
	if !clicked {
		return nil, fmt.Errorf(`"Edit release" button was missing or disabled`)
	}
	prepareReady := formHelpers + ` && ` + rolloutHelpers + ` && !!window.__gplayRollout.nextButton()`
	if err := b.EvalUntil(ctx, prepareReady, 60*time.Second); err != nil {
		return nil, fmt.Errorf("release prepare page did not load: %w", err)
	}
	var state PrepareState
	if err := b.Eval(ctx, readPrepareScript, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

const nextClickScript = `(() => {
  const r = window.__gplayRollout;
  const btn = r && r.nextButton();
  if (!r.enabled(btn)) return false;
  btn.click();
  return true;
})()`

// ReviewState is the release review (Preview and confirm) page state.
type ReviewState struct {
	Warnings []string `json:"warnings"`
	CanSave  bool     `json:"canSave"`
}

const readReviewScript = `(() => {
  const r = window.__gplayRollout;
  return {
    warnings: r.warnings(),
    canSave: r.enabled(r.saveButton()),
  };
})()`

// ReviewDraftRelease advances from the prepare page to the review page and
// reads it.
func ReviewDraftRelease(ctx context.Context, b *Browser) (*ReviewState, error) {
	var clicked bool
	if err := b.Eval(ctx, formHelpers+` && `+rolloutHelpers+` && `+nextClickScript, &clicked); err != nil {
		return nil, err
	}
	if !clicked {
		return nil, fmt.Errorf(`"Next" button was missing or disabled`)
	}
	reviewReady := formHelpers + ` && ` + rolloutHelpers + ` && !!window.__gplayRollout.saveButton()`
	if err := b.EvalUntil(ctx, reviewReady, 60*time.Second); err != nil {
		return nil, fmt.Errorf("release review page did not load: %w", err)
	}
	var state ReviewState
	if err := b.Eval(ctx, readReviewScript, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

const saveReviewClickScript = `(() => {
  const r = window.__gplayRollout;
  const btn = r && r.saveButton();
  if (!r.enabled(btn)) return false;
  btn.click();
  return true;
})()`

const publishingOverviewPresentScript = `(() =>
  !!document.querySelector('[debug-id=not-sent-for-review-changes]'))()`

// SaveReleaseReview saves the reviewed release, staging it in the Publishing
// overview, and waits for the overview to show it.
func SaveReleaseReview(ctx context.Context, b *Browser, timeout time.Duration) error {
	var clicked bool
	if err := b.Eval(ctx, formHelpers+` && `+rolloutHelpers+` && `+saveReviewClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Save" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, publishingOverviewPresentScript, timeout); err != nil {
		return fmt.Errorf("release change did not reach the Publishing overview: %w", err)
	}
	return nil
}
