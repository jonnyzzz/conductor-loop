//go:build e2e

package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMonitoringUI(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("set E2E=1 to run Playwright UI test")
	}

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root = filepath.Dir(filepath.Dir(root))

	spec := `const { test, expect } = require('@playwright/test');

test('monitoring UI flow', async ({ page }) => {
  const baseURL = process.env.PLAYWRIGHT_TEST_BASE_URL || 'http://localhost:5173';
  await page.goto(baseURL);
  await expect(page.getByText('Conductor Loop Monitor')).toBeVisible();

  const firstProject = page.locator('.list-item').first();
  if (await firstProject.count()) {
    await firstProject.click();
  }

  const taskButtons = page.locator('.list-item').filter({ hasText: 'tasks' });
  if (await taskButtons.count()) {
    await taskButtons.first().click();
  }

  await expect(page.getByText('Live logs')).toBeVisible();
});`

	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "ui.spec.js")
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cmd := exec.Command("npx", "playwright", "test", specPath)
	cmd.Dir = root
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("playwright test failed: %v\n%s", err, string(output))
	}
}
