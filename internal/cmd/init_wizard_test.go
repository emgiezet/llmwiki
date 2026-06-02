package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/wizard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteProjectConfig_MinimalNoForce(t *testing.T) {
	dir := t.TempDir()
	err := writeProjectConfig(dir, initOptions{customer: "acme", projectType: "client"}, false)
	require.NoError(t, err)

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "acme", cfg.Customer)
	assert.Equal(t, "client", cfg.Type)
	assert.Equal(t, "", cfg.Extraction.Preset)
	assert.Equal(t, "", cfg.OutputMode)
}

func TestWriteProjectConfig_NoForce_Errors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeProjectConfig(dir, initOptions{projectType: "client"}, false))
	err := writeProjectConfig(dir, initOptions{projectType: "personal"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestWriteProjectConfig_Force_Overwrites(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeProjectConfig(dir, initOptions{projectType: "client"}, false))
	err := writeProjectConfig(dir, initOptions{projectType: "personal"}, true)
	require.NoError(t, err)

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "personal", cfg.Type)
}

func TestWriteProjectConfig_WritesPresetAndOutputMode(t *testing.T) {
	dir := t.TempDir()
	err := writeProjectConfig(dir, initOptions{
		projectType:  "personal",
		preset:       "notes",
		outputMode:   "both",
		localDocsDir: "docs/llmwiki",
	}, false)
	require.NoError(t, err)

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "notes", cfg.Extraction.Preset)
	assert.Equal(t, "both", cfg.OutputMode)
	assert.Equal(t, "docs/llmwiki", cfg.LocalDocsDir)
}

func TestInstallIntegrations_PreCommitOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))

	installIntegrations(dir, false, true)

	if _, err := os.Stat(filepath.Join(dir, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("pre-commit hook not installed: %v", err)
	}
}

func TestInstallIntegrations_None(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))

	installIntegrations(dir, false, false)

	if _, err := os.Stat(filepath.Join(dir, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook should not exist; err=%v", err)
	}
}

func TestRunInitWizard_ClientProject(t *testing.T) {
	orig := graymatterDetected
	graymatterDetected = func() bool { return false }
	defer func() { graymatterDetected = orig }()

	// type=client(1), customer=acme, preset=notes(6), output=both(3), localdir=Enter, preCommit=n, save=y
	input := "1\nacme\n6\n3\n\nn\ny\n"
	p := wizard.New(strings.NewReader(input), &bytes.Buffer{})

	opts, inst, save := runInitWizard(p, config.ProjectConfig{})

	assert.True(t, save)
	assert.Equal(t, "client", opts.projectType)
	assert.Equal(t, "acme", opts.customer)
	assert.Equal(t, "notes", opts.preset)
	assert.Equal(t, "both", opts.outputMode)
	assert.Equal(t, "docs/llmwiki", opts.localDocsDir)
	assert.False(t, inst.preCommit)
}

func TestRunInitWizard_PersonalNoCustomer(t *testing.T) {
	orig := graymatterDetected
	graymatterDetected = func() bool { return false }
	defer func() { graymatterDetected = orig }()

	// type=personal(2) → no customer prompt; preset=Enter(default), output=Enter(central), preCommit=n, save=n
	input := "2\n\n\nn\nn\n"
	p := wizard.New(strings.NewReader(input), &bytes.Buffer{})

	opts, _, save := runInitWizard(p, config.ProjectConfig{})

	assert.False(t, save)
	assert.Equal(t, "personal", opts.projectType)
	assert.Equal(t, "", opts.customer)
	assert.Equal(t, "default", opts.preset)
	assert.Equal(t, "central", opts.outputMode)
}

func TestAnyInitFlagChanged(t *testing.T) {
	cmd := NewInitCmd()
	assert.False(t, anyInitFlagChanged(cmd))

	require.NoError(t, cmd.Flags().Set("customer", "acme"))
	assert.True(t, anyInitFlagChanged(cmd))
}

func TestInitCmd_NonInteractiveWithFlag_WritesConfig(t *testing.T) {
	orig := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = orig }()

	dir := t.TempDir()
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--type", "personal", "--no-graymatter", dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	require.NoError(t, cmd.Execute())

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "personal", cfg.Type)
}

func TestInitCmd_InteractiveNoFlags_RunsWizard(t *testing.T) {
	origI := isInteractive
	isInteractive = func() bool { return true }
	defer func() { isInteractive = origI }()
	origG := graymatterDetected
	graymatterDetected = func() bool { return false }
	defer func() { graymatterDetected = origG }()

	dir := t.TempDir()
	cmd := NewInitCmd()
	cmd.SetArgs([]string{dir})
	// type=personal(2), preset=Enter, output=Enter(central), preCommit=n, save=y
	cmd.SetIn(strings.NewReader("2\n\n\nn\ny\n"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	require.NoError(t, cmd.Execute())

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "personal", cfg.Type)
}

func TestInitCmd_InteractiveEditMode_WarnsOverwrite(t *testing.T) {
	origI := isInteractive
	isInteractive = func() bool { return true }
	defer func() { isInteractive = origI }()
	origG := graymatterDetected
	graymatterDetected = func() bool { return false }
	defer func() { graymatterDetected = origG }()

	dir := t.TempDir()
	// Pre-existing config makes this an edit, not a fresh init.
	require.NoError(t, writeProjectConfig(dir, initOptions{projectType: "client", customer: "acme"}, false))

	out := &bytes.Buffer{}
	cmd := NewInitCmd()
	cmd.SetArgs([]string{dir})
	// type=personal(2), preset=Enter, output=Enter, preCommit=n, save=y
	cmd.SetIn(strings.NewReader("2\n\n\nn\ny\n"))
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, out.String(), "overwrite")
}
