package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzDetectServices(f *testing.F) {
	f.Add([]byte(`services:
  web:
    image: nginx
  db:
    image: postgres`))
	f.Add([]byte(`services:
  "../../../tmp":
    image: x`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), data, 0644); err != nil {
			t.Fatal(err)
		}
		_, _ = DetectServices(dir)
	})
}
