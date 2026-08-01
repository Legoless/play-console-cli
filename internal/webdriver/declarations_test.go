package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReadDeclarations(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(formHelpers+` && !!document.querySelector('[debug-id=tab-strip]') &&
	  !! [...document.querySelectorAll('[role=tab]')].find(t => /actioned/i.test(t.textContent))`, true)
	f.setReply(`(/all caught up/i.test(document.body.innerText || '')) ||
	  !!document.querySelector('[debug-id^=policy-declaration-id-][debug-id$=-section]')`, true)
	f.setReply(readAttentionScript, []map[string]any{})
	f.setReply(openActionedTabScript, true)
	f.setReply(`!!document.querySelector('[debug-id^=policy-declaration-id-][debug-id$=-section]')`, true)
	f.setReply(readDeclarationsScript, map[string]any{
		"actioned": []map[string]any{
			{"key": "privacy-policy", "title": "Privacy policy", "lastEdited": "Jul 29, 2026"},
			{"key": "psl", "title": "Data safety", "lastEdited": "Jul 29, 2026"},
		},
	})
	b := connectFake(t, f)

	state, err := ReadDeclarations(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com")
	if err != nil {
		t.Fatalf("ReadDeclarations: %v", err)
	}
	if len(state.Actioned) != 2 {
		t.Fatalf("actioned = %+v", state.Actioned)
	}
	if state.Actioned[0].Key != "privacy-policy" || state.Actioned[0].LastEdited == "" {
		t.Errorf("first = %+v", state.Actioned[0])
	}
	if len(state.NeedAttention) != 0 {
		t.Errorf("needAttention = %+v, want none when caught up", state.NeedAttention)
	}
}

func TestReadDeclarations_FlagsNeedingAttention(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(formHelpers+` && !!document.querySelector('[debug-id=tab-strip]') &&
	  !! [...document.querySelectorAll('[role=tab]')].find(t => /actioned/i.test(t.textContent))`, true)
	f.setReply(`(/all caught up/i.test(document.body.innerText || '')) ||
	  !!document.querySelector('[debug-id^=policy-declaration-id-][debug-id$=-section]')`, true)
	f.setReply(readAttentionScript, []map[string]any{{"key": "psl", "title": "Data safety", "needsAction": true}})
	f.setReply(openActionedTabScript, true)
	f.setReply(`!!document.querySelector('[debug-id^=policy-declaration-id-][debug-id$=-section]')`, true)
	f.setReply(readDeclarationsScript, map[string]any{"actioned": []map[string]any{}})
	b := connectFake(t, f)

	state, err := ReadDeclarations(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com")
	if err != nil {
		t.Fatalf("ReadDeclarations: %v", err)
	}
	if len(state.NeedAttention) == 0 || !state.NeedAttention[0].NeedsAction {
		t.Errorf("needAttention = %+v, want a needs-action entry", state.NeedAttention)
	}
}

func radioFormReadyExpr() string {
	return declarationFormHelpers + ` && !!window.__gplayDecl.bar()`
}

func TestReadRadioDeclaration(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(radioFormReadyExpr(), true)
	f.setReply(declarationFormHelpers+` && `+readRadioDeclarationScript, map[string]any{
		"answer": "no", "canSave": true,
	})
	b := connectFake(t, f)

	state, err := ReadRadioDeclaration(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com", "government-apps")
	if err != nil {
		t.Fatalf("ReadRadioDeclaration: %v", err)
	}
	if state.Answer != "no" || !state.CanSave {
		t.Errorf("state = %+v", state)
	}
}

func TestSetRadioDeclaration(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(radioFormReadyExpr(), true)
	f.setReply(declarationFormHelpers+` && (() => {
  const r = window.__gplayDecl.radios();
  if (r.length < 2) return 'no-radios';
  r[1].click();
  return 'clicked';
})()`, "clicked")
	f.setReply(declarationFormHelpers+` && (() => {
  const r = window.__gplayDecl.radios();
  if (r.length < 2) return false;
  const c = r[1].getAttribute('aria-checked');
  return c === 'true' || r[1].checked === true;
})()`, true)
	f.setReply(declarationFormHelpers+` && `+submitDeclarationClickScript, true)
	f.setReply(declarationSaveSettledScript, true)
	// The current answer is "yes"; after the save the persisted answer is "no".
	f.setReply(declarationFormHelpers+` && `+readRadioDeclarationScript, map[string]any{
		"answer": "yes", "canSave": true,
	})
	go func() {
		time.Sleep(200 * time.Millisecond)
		f.setReply(declarationFormHelpers+` && `+readRadioDeclarationScript, map[string]any{
			"answer": "no", "canSave": true,
		})
	}()
	b := connectFake(t, f)

	changed, err := SetRadioDeclaration(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com", "government-apps", false)
	if err != nil {
		t.Fatalf("SetRadioDeclaration: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true")
	}
}

func TestSetRadioDeclaration_NoOpWhenAlreadySet(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(radioFormReadyExpr(), true)
	f.setReply(declarationFormHelpers+` && `+readRadioDeclarationScript, map[string]any{
		"answer": "no", "canSave": true,
	})
	b := connectFake(t, f)

	changed, err := SetRadioDeclaration(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com", "government-apps", false)
	if err != nil {
		t.Fatalf("SetRadioDeclaration: %v", err)
	}
	if changed {
		t.Error("changed = true, want false for a no-op")
	}
}

func TestSetRadioDeclaration_ErrorsWhenNotSaved(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(radioFormReadyExpr(), true)
	f.setReply(declarationFormHelpers+` && (() => {
  const r = window.__gplayDecl.radios();
  if (r.length < 2) return 'no-radios';
  r[1].click();
  return 'clicked';
})()`, "clicked")
	f.setReply(declarationFormHelpers+` && (() => {
  const r = window.__gplayDecl.radios();
  if (r.length < 2) return false;
  const c = r[1].getAttribute('aria-checked');
  return c === 'true' || r[1].checked === true;
})()`, true)
	f.setReply(declarationFormHelpers+` && `+submitDeclarationClickScript, true)
	f.setReply(declarationSaveSettledScript, true)
	f.setReply(declarationFormHelpers+` && `+readRadioDeclarationScript, map[string]any{
		"answer": "yes", "canSave": true,
	})
	b := connectFake(t, f)

	_, err := SetRadioDeclaration(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com", "government-apps", false)
	if err == nil || !strings.Contains(err.Error(), "was not saved") {
		t.Errorf("err = %v, want not-saved error", err)
	}
}

func TestReadPolicyStatus_Empty(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(`!!document.querySelector('[debug-id=empty-state], [role=row]')`, true)
	f.setReply(readPolicyStatusScript, map[string]any{
		"state": "empty", "message": "Information about your compliance will be shown here after your app has been reviewed", "issues": []map[string]any{},
	})
	b := connectFake(t, f)

	status, err := ReadPolicyStatus(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com")
	if err != nil {
		t.Fatalf("ReadPolicyStatus: %v", err)
	}
	if status.State != "empty" || status.Message == "" || len(status.Issues) != 0 {
		t.Errorf("status = %+v", status)
	}
}

func TestReadPolicyStatus_Issues(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(`!!document.querySelector('[debug-id=empty-state], [role=row]')`, true)
	f.setReply(readPolicyStatusScript, map[string]any{
		"state": "issues", "issues": []map[string]any{{"title": "Content rating", "detail": "Rejected | resubmit"}},
	})
	b := connectFake(t, f)

	status, err := ReadPolicyStatus(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com")
	if err != nil {
		t.Fatalf("ReadPolicyStatus: %v", err)
	}
	if status.State != "issues" || len(status.Issues) != 1 || status.Issues[0].Title != "Content rating" {
		t.Errorf("status = %+v", status)
	}
}

func TestPublishApprovedChanges_NoAction(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(formHelpers+` && `+publishHelpers+` && !!document.querySelector('[debug-id=managed-publishing-dropdown]')`, true)
	f.setReply(publishNowClickScript, false)
	b := connectFake(t, f)

	err := PublishApprovedChanges(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com", 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "nothing to publish") {
		t.Errorf("err = %v, want nothing-to-publish error", err)
	}
}

func TestPublishApprovedChanges_Confirms(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(formHelpers+` && `+publishHelpers+` && !!document.querySelector('[debug-id=managed-publishing-dropdown]')`, true)
	f.setReply(publishNowClickScript, true)
	f.setReply(sendForReviewDialogPresentScript, true)
	f.setReply(sendForReviewConfirmScript, "confirmed")
	f.setReply(`(() => ![...document.querySelectorAll('button')].find(b =>
	  b.getClientRects().length && /^publish/i.test(b.textContent.trim()))())()`, true)
	b := connectFake(t, f)

	if err := PublishApprovedChanges(context.Background(), b, publishingTestDeveloper, publishingTestApp, "me@example.com", 2*time.Second); err != nil {
		t.Fatalf("PublishApprovedChanges: %v", err)
	}
}
