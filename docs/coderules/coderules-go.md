# Code Rules and Processes for this project

**KEY RULES**:

- NEVER CLAIM THAT SOMETHING IS COMPLETE IF THERE ARE REGRESSIONS. RUN THE FULL TEST SUITE BEFORE AND AFTER EACH WORK UNIT.
- **ALL CODE MUST PASS `go vet ./...` AND `golangci-lint run`** before committing

## Test Locations

- **Integration tests go in `tests/`** or alongside source files in `*_test.go` files — use `*_test.go` alongside source by default
- **Internal tests** (testing unexported functions) belong in `*_test.go` files in the same package
- **Black-box tests** (testing only exported API) use the `_test` package suffix (e.g., `package foo_test`)
- Test functions must be named `Test*` and accept `*testing.T`

## Developing in Go

- Never do "fallback" programming in terms of requirements: if you expect a dependency, fail immediately if it's not present
- Use standard `errors` and `fmt.Errorf` with `%w` for error wrapping
- **Error handling**:
  - Return `error` as the last return value for recoverable errors; never panic for expected failure cases
  - Avoid `log.Fatal` and `os.Exit` in library code; acceptable in `main` and tests
  - Check and handle every returned error; do not discard with `_`
  - Propagate errors with context using `fmt.Errorf("context: %w", err)`
- Target Go 1.22+ (use latest stable)
- Backwards compatibility is NOT a goal, neither in terms of Go, nor in terms of this project's code itself
- Prefer explicit types over excessive type inference where it aids readability
- Use Go modules (`go.mod`) for dependency management

## Code Quality

- **CRITICAL**: Run `golangci-lint run ./...` before committing; treat all warnings as errors
- Run `go vet ./...` before committing
- Run `gofmt -l .` (or `goimports`) before committing to ensure consistent formatting — there must be no output
- Prefer idiomatic Go: use `range`, slices, maps; avoid unnecessary abstraction
- Use `const` for compile-time constants
- Avoid `unsafe` unless absolutely necessary; document why it's safe if used

## Debugging

- **CRITICAL**: Always identify root causes of failures. Do NOT treat the symptoms of failures.
- Use `fmt.Println` or `log.Println` for quick debugging, but remove before committing
- Use structured logging (e.g., `slog`) for permanent logging needs

## Process

- Don't back files up by copying! We use git for versioning.
- For each new development stage, create a new git branch first.
- We practice TDD:
  - Write tests first that demonstrate the desired behaviour
  - Pause for human review of the tests
  - Progress the implementation until the tests succeed.
  - NEVER tweak a test to "fit" the behaviour, unless the test is demonstrably broken.
  - Once a test set has been reviewed and approved, that's a contract: do NOT skip or change without re-approval. All approved tests MUST pass before PR.
  - Before opening a PR, you MUST ensure that the full test suite is green.
  - Review any `t.Skip()` tests and ensure they are documented.
  - Fix any compiler warnings and linter issues.
- Maintain progress in docs/TODO-X.md files
- Don't use /tmp and other locations outside the current repository: use the tmp/ directory in the repository dir instead, provided for this purpose
- If you create temporary scripts for debugging, remove them after use, and ensure not committed to git

## Git Workflow


3. **Write tests FIRST** - request a test-only adversarial review from a sub-agent
4. **Commit approved tests** - Only after review approval
5. **Implement until tests pass** - Make the tests green
6. **Run complete test suite** - No regressions tolerated!
9. **Await implementation review** - request a adversarial review of the full implementation from a sub-agent

In any git and GitHub messaging (commit messages, PR messages, issues, comments etc), we maintain a terse, professional tone:

1. **Never make unproven claims**: don't make claims about the validity, effectiveness or awesomeness of your changes in a commit or other message. By definition, that is determined by the CI results, which you can't see yet. Explain what was done, and why. Be modest and factual.
2. **Never use emoji symbols**: we're not 14-year-olds on Instagram here. No green ticks, no red crosses, no smileys, no symbols.
3. **Don't use bold text**: don't embellish or add emphasis with bold or italic text.
4. **Brevity**: issues and commit messages are written for co-workers. Respect their time. Obviously, be complete, but express yourself in a professional, concise tone.
5. **UK English**: we use UK English spelling throughout.
