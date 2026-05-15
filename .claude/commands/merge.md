# /merge - merge completed work unit and tidy up

## Pre-merge Checks (MANDATORY - DO NOT SKIP)

Before ANY merge to main, you MUST verify:

1. `gofmt -l .` passes
2. `go vet ./... && golangci-lint run` passes
3. `go test ./...` passes (full test suite)

If any check fails, DO NOT MERGE. Fix the issues first.

## Merge Process

Only after all checks pass:

1. Commit any uncommitted work
2. Merge the current branch into `main`
3. Delete the spent local branch
4. Close the GitHub issue that you completed
5. Check the matching Epic, if relevant. If this issue was the last in the Epic, also close the Epic.
6. Re-read @docs/coderules/coderules-go.md

If there are remaining issues in the Epic, suggest the next issue to work on to the user.
