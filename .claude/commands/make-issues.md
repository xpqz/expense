# /make-issues - Convert a plan document to actionable issues

Carefully examine the plan document given in $ARGUMENTS (e.g., "abc" assumed to live under "docs/plans" and have a ".md" extension if not given).

Draw up GitHub epic and linked issues for the plan document, ensuring that every issue references the relevant section in the design document explicitly for context. Ensure issues are created in the most logical implementation order, noting dependencies. Report a summary to the user listing every created issue, starting with the Epic(s). Propose the most logical issue to begin work on.

## Example output

Created epic #312 with 5 linked issues:
┌──────┬──────┬──────────────────────────────────────────────────┬────────────────────┐
│ ID │ Type │ Title │ Phase │
├──────┼──────┼──────────────────────────────────────────────────┼────────────────────┤
│ #312 │ epic │ Implementation of feature ABC│ - │
├──────┼──────┼──────────────────────────────────────────────────┼────────────────────┤
│ #313 │ task │ ABC setup │ Phase 1a │
├──────┼──────┼──────────────────────────────────────────────────┼────────────────────┤
│ #314 │ task │ ABC sub-feature A │ Phase 1b │
├──────┼──────┼──────────────────────────────────────────────────┼────────────────────┤
│ #315 │ task │ ABC sub-feature B │ Phase 2 │
└──────┴──────┴──────────────────────────────────────────────────┴────────────────────┘
Each issue references specific sections in docs/plans/abc.md including semantics, edge cases, error conditions, and test plans. Issues are linked sequentially to reflect implementation dependencies.

I propose we begin with Issue #313.
