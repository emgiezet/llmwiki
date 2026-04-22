package cmd_test

import (
	"io"
	"os"
	"testing"

	"github.com/mgz/llmwiki/internal/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "llmwiki"}
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.AddCommand(cmd.NewAbsorbCmd(), cmd.NewHookCmd())
	return root
}

func TestAbsorbCmd_NoteStdin_DoesNotErrorWhenMemoryDisabled(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	go func() {
		io.WriteString(w, "Claude discovered the retry mechanism uses exponential back-off with jitter")
		w.Close()
	}()

	root := buildTestRoot()
	root.SetArgs([]string{"absorb", t.TempDir(), "--note-stdin"})
	// Memory disabled in temp env → warning printed but no error.
	assert.NoError(t, root.Execute())
}

func TestAbsorbCmd_NoteStdin_MutuallyExclusiveWithNote(t *testing.T) {
	root := buildTestRoot()
	root.SetArgs([]string{"absorb", t.TempDir(), "--note", "manual note", "--note-stdin"})
	err := root.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot combine")
}

func TestAbsorbCmd_NoteStdin_RejectsOversizedInput(t *testing.T) {
	const twoMiB = 2 << 20

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	// Write 2 MiB of zeros then close the pipe.
	go func() {
		zeros := make([]byte, twoMiB)
		w.Write(zeros) //nolint:errcheck
		w.Close()
	}()

	root := buildTestRoot()
	root.SetArgs([]string{"absorb", t.TempDir(), "--note-stdin"})
	err = root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
	assert.Contains(t, err.Error(), "1048576")
}
