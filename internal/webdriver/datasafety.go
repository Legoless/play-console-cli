package webdriver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// importDialogPresentScript reports whether the Import-from-CSV dialog is up.
const importDialogPresentScript = `(() => {
  const d = document.querySelector('.pane.modal.visible, [role=dialog]');
  return !!(d && /import from csv/i.test(d.innerText || ''));
})()`

const importDialogHasFileInputScript = `(() =>
  !!document.querySelector('.pane.modal.visible input[type=file], [role=dialog] input[type=file]'))()`

const importDialogClickImportScript = `(() => {
  const d = document.querySelector('.pane.modal.visible') ||
    [...document.querySelectorAll('[role=dialog]')].find(d => d.getClientRects().length);
  if (!d) return 'no-dialog';
  const btn = [...d.querySelectorAll('button')].find(b =>
    /^import$/i.test(b.textContent.trim()) && !b.disabled);
  if (!btn) return 'no-import';
  btn.click();
  return 'clicked';
})()`

const importDialogGoneScript = `(() =>
  !document.querySelector('.pane.modal.visible input[type=file], [role=dialog] input[type=file]'))()`

// ImportDataSafetyCSV imports a data safety answers CSV into the
// data-privacy-security questionnaire. It overwrites any answers already in
// the form; callers must walk the wizard to Preview and save (see
// QuestionnaireNext/QuestionnaireSave).
func ImportDataSafetyCSV(ctx context.Context, b *Browser, developerID, appID, account, filePath string) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("CSV file: %w", err)
	}
	if err := b.Navigate(ctx, appContentURL(developerID, appID, account, "data-privacy-security")); err != nil {
		return err
	}
	ready := formHelpers + ` && !!document.querySelector('[debug-id=button-import-csv]')`
	if err := b.EvalUntil(ctx, ready, 60*time.Second); err != nil {
		return fmt.Errorf("data safety page did not load (is the gplay browser profile signed in?): %w", err)
	}
	var clicked bool
	if err := b.Eval(ctx, `(() => {
	  const w = document.querySelector('[debug-id=button-import-csv]');
	  const btn = w && (w.querySelector('button') || w);
	  if (!btn || btn.disabled) return false;
	  btn.click();
	  return true;
	})()`, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Import from CSV" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, importDialogPresentScript+` && `+importDialogHasFileInputScript, 30*time.Second); err != nil {
		return fmt.Errorf("the import dialog did not open: %w", err)
	}
	if err := b.SetFileInputFiles(ctx, ".pane.modal.visible input[type=file], [role=dialog] input[type=file]", []string{filePath}); err != nil {
		return err
	}
	var result string
	if err := b.Eval(ctx, importDialogClickImportScript, &result); err != nil {
		return err
	}
	if result != "clicked" {
		return fmt.Errorf("the import dialog's Import button was missing or disabled (state %q)", result)
	}
	if err := b.EvalUntil(ctx, importDialogGoneScript, 60*time.Second); err != nil {
		return fmt.Errorf("the import did not complete: %w", err)
	}
	return nil
}
