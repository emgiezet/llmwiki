.PHONY: build test security-scan sbom clean

build:
	go build -o llmwiki .

test:
	go test ./...

# Runs the full security gate. Requires the following tools in $PATH:
#   staticcheck, gosec, govulncheck, osv-scanner, gitleaks
# Install with:
#   go install honnef.co/go/tools/cmd/staticcheck@latest
#   go install github.com/securego/gosec/v2/cmd/gosec@latest
#   go install golang.org/x/vuln/cmd/govulncheck@latest
#   go install github.com/google/osv-scanner/cmd/osv-scanner@latest
#   go install github.com/zricethezav/gitleaks/v8@latest
security-scan:
	@echo "==> go vet"
	go vet ./...
	@echo "==> staticcheck"
	staticcheck ./...
	@echo "==> gosec"
	# G304 (file path from variable) and G204 (subprocess with variable) are excluded because
	# llmwiki is a file-scanner that intentionally reads user-supplied paths and shells out
	# to git/claude. Residual risks documented in docs/threat-model.md.
	gosec -exclude-dir=testdata -exclude=G304,G204 ./...
	@echo "==> govulncheck"
	govulncheck ./...
	@echo "==> osv-scanner"
	osv-scanner --lockfile=go.mod
	@echo "==> gitleaks"
	gitleaks detect --source=. --redact --log-opts="--all"

sbom:
	cyclonedx-gomod mod -json -output sbom.cdx.json

clean:
	rm -f llmwiki
