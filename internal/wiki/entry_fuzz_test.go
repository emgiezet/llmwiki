package wiki

import "testing"

func FuzzParseProjectEntry(f *testing.F) {
	f.Add([]byte(`---
name: foo
customer: acme
---
body`))
	f.Add([]byte(`---
tags: [a, b, c]
---
body`))
	f.Add([]byte(``))
	f.Add([]byte(`not yaml at all`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseProjectEntry(data)
	})
}

func FuzzParseServiceEntry(f *testing.F) {
	f.Add([]byte(`---
name: svc
---
body`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseServiceEntry(data)
	})
}

func FuzzParseClientEntry(f *testing.F) {
	f.Add([]byte(`---
customer: acme
---
body`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseClientEntry(data)
	})
}

func FuzzParseMultiProjectEntry(f *testing.F) {
	f.Add([]byte(`---
name: p
---
body`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseMultiProjectEntry(data)
	})
}
