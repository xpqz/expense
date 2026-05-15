# /crev - Code Review for Issue

Review the implementation for issue $ARGUMENTS (e.g., "301" or "309").

## Process

We practice test-driven development, and so review in two stages:

1. Tests only -- the first review will contain tests only, no implementation. No need to run the test set, as they are expected to fail. Examine the test surface and the implied API.
2. Implementation -- validate the implementation carefully, see below.

Bug fixes represent a special case: if the issue to be fixed is a bug, you **MUST** validate that the fix goes to the root cause of the problem, not a symptom "workaround". Is there a set of tests that demonstrate the bug reliably, so we can protect against regressions? Does the implementation demonstrate that the root cause of the problem has been determined?

### 1. Gather Context

Read in full the following sources to understand the issue:

- `docs/prs/$ARGUMENTS.md` - PR documentation, if present
- GitHub issue as defined by $ARGUMENTS. Use "gh"
- Any design documents referenced in the PR doc (e.g., `docs/plans/*.md`, `docs/bugs/*.md`)

### 2. Implementation Review

Examine the implementation for:

#### Architectural conformity

- Read and understand the relevant documents and sections under the "docs/plans/" tree for context.

#### Code Quality

- **File organisation**: Are changes in logical locations? Are any files becoming too long (>1000 lines)?
- **API consistency**: Are public APIs consistent with existing patterns? Are private abstractions leaking?
- **Naming**: Are functions and variables named clearly and consistently?
- **Comments**: Do comments accurately describe the code? Are complex algorithms documented?
- **Error handling**: Are errors handled appropriately with proper error types?

### 3. Run Verification

Execute the following checks, unless we're reviewing tests prior to implementation:

```bash
go test ./...                                # Full test suite
gofmt -l .                                   # Formatting
go vet ./... && golangci-lint run            # Build and lint checks
```

Note:

- Regressions are NOT tolerated - all existing tests must pass
- Under TDD, new tests may fail if reviewing tests before implementation

### 4. Write Review

Create or update if already present `docs/reviews/$ARGUMENTS.md` with findings organised as:

```markdown
# Review: Issue #$ARGUMENTS - [Brief Title]

## Summary

[One paragraph overview of findings]

## Findings

### [Severity]: [Finding Title]

[Description of the issue and recommendation]

Location: [file:line or general area]

### ...

## Verification

- Tests: [pass/fail/not run]
- Build/analyser checks: [clean/warnings]
- Format: [clean/issues]

## Recommendation

[Approve / Approve with minor changes / Request changes]
```

Severity levels:

- **Critical**: Must fix before merge (correctness issues, regressions)
- **Major**: Should fix before merge (significant gaps, API issues)
- **Minor**: Nice to fix (style, documentation, minor improvements)
- **Note**: Observations that don't require changes

If no issues found, a brief "No issues found" summary is acceptable.
