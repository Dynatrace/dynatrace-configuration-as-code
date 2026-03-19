  Title: Add option to print rendered payloads during deploy / --dry-run

  Body:
  Monaco currently renders templates during deploy --dry-run, but it doesn’t expose the final JSON payloads. Please add a flag like --print-rendered-payloads or render subcommand to show the resolved request body per config ID/project. This would help debugging templates, reviewing changes, and validating Settings/API payloads before deployment. Redaction for secrets would be helpful.

  Contribution setup

   - fork repo
   - create issue first
   - branch: feature/{Issue}/{description}
   - build: go build ./cmd/monaco
   - test: go generate ./... then go test -tags=unit -v -race ./...
