# Skills Workflow Experiment: Lean Context + Self-Reflection + Ruthless Review

**Date:** 2025-11-26
**Context:** Implementing devcontainer test coverage (10 tasks, ~770 LOC)
**Baseline:** Standard `superpowers:subagent-driven-development` skill

---

## What We Tried Differently

### 1. Lean Context for Implementers

**Standard approach:** Give subagent the full plan file to read.

**Experiment:** Give subagent ONLY what they need:
- The specific task (1-2 sentences)
- The pattern to follow (reference to existing code)
- The exact file to modify
- The test command to run

**Example prompt (Task 1):**
```
You are adding a single E2E test to packnplay's test suite.

**Your task:** Add `TestE2E_FeaturePrivilegedMode` to `pkg/runner/e2e_test.go`

**What to test:** A local devcontainer feature that requests `"privileged": true`
in its metadata should result in the container running with `--privileged` flag.

**Follow the exact pattern of TestE2E_FeatureOptionValidation** (at the end of the file)

**After writing, run:** `go test -v ./pkg/runner -run TestE2E_FeaturePrivilegedMode -timeout 5m`
```

### 2. Self-Reflection Before Handoff

**Standard approach:** Subagent reports completion, moves to next task.

**Experiment:** Added to every implementer prompt:
```
When done, look at your work with fresh eyes and tell me:
how could it have been better, or what needs improvement?
```

### 3. Separate Ruthless Code Reviewer

**Standard approach:** Single code review at end or between major phases.

**Experiment:** After each task:
1. Implementer completes + self-reflects
2. Separate subagent reviews with prompt: "Be ruthless. Find problems."
3. Fix issues before moving on

---

## What Worked Well

### Lean Context Produced Faster, More Focused Work
- Subagents didn't waste tokens reading irrelevant plan sections
- Clear pattern reference ("follow TestE2E_FeatureOptionValidation") was more effective than abstract instructions
- Tasks completed in single attempts more often

### Self-Reflection Surfaced Real Issues
The implementer for Task 5 (entrypoint) identified through self-reflection that their test was failing because of an implementation bug, not a test bug. They traced it to line 99 of runner.go where `strings.Join(metadata.Entrypoint, " ")` was creating invalid Docker entrypoint syntax.

Without the self-reflection prompt, they might have just reported "test fails" without the root cause analysis.

### Ruthless Reviewer Caught Critical Bug
Task 1's implementer fixed the path resolution issue. But the ruthless reviewer found:

> **CRITICAL: Line 982 uses `workDir` instead of `mountPath`** - The fix was applied in two places but with DIFFERENT base directories. This will cause features to fail to resolve when `mountPath != workDir`.

This would have caused subtle bugs in worktree scenarios. The implementer didn't catch it because they were focused on the immediate fix, not the broader consistency.

### Bugs Found by This Workflow
1. **Path resolution inconsistency** (mountPath vs workDir) - caught by reviewer
2. **Entrypoint join bug** (array joined with spaces) - caught by self-reflection
3. **Test assertion bug** (checking wrong Docker field) - caught during fix iteration

---

## What Needs Improvement

### Reviewer File Access
The first code reviewer couldn't find the test file:
> "The file doesn't appear to exist in the repository, or I'm unable to access it"

**Fix needed:** Reviewer prompts should explicitly say "Read file X first" rather than assuming the subagent will find it.

### Test-Then-Fix Flow Overhead
When a test reveals an implementation bug:
- Current: Implementer reports → I dispatch fixer → fixer fixes → I verify
- Better: Let implementer fix if they identified the root cause in self-reflection

The extra round-trip added latency without adding value when the implementer already knew the fix.

### Docker Availability Handling
Several tests skipped because Docker became unresponsive mid-session. The workflow should:
- Verify Docker early and fail fast
- Have a "Docker unavailable" mode that at least verifies code compiles

### Parallel Task Batching
Tasks 3-5 (securityOpt, init, entrypoint) were very similar. Running them in parallel worked well, but the prompts were nearly identical. Could have a "batch similar tasks" pattern.

---

## Recommended Skill Changes

### For `subagent-driven-development`:

1. **Add lean context option:**
   ```
   Instead of having subagent read full plan file, extract only:
   - Task description
   - Files to modify
   - Pattern to follow
   - Verification command
   ```

2. **Add self-reflection step:**
   ```
   After task completion, require subagent to answer:
   "Look at your work with fresh eyes - what could be better
   or needs improvement?"
   ```

3. **Add ruthless review step:**
   ```
   Dispatch separate reviewer with explicit instructions:
   - "Be ruthless. Find problems."
   - "Read [specific file] first"
   - "Check for: bugs, edge cases, consistency with patterns"
   ```

4. **Allow implementer to fix self-identified issues:**
   ```
   If self-reflection identifies a fixable issue, let same
   subagent fix it before handoff to reviewer.
   ```

### For `requesting-code-review`:

1. **Explicit file reading:**
   ```
   Always start reviewer prompt with:
   "First, read [file path]. Then review for..."
   ```

2. **Confidence levels:**
   The reviewer in our experiment used confidence levels (100%, 95%, 90%, etc.) which was helpful for prioritizing fixes. Consider making this standard.

---

## Metrics

| Metric | Value |
|--------|-------|
| Tasks completed | 8 of 10 |
| Bugs caught by self-reflection | 1 |
| Bugs caught by ruthless review | 1 |
| Bugs caught during implementation | 1 |
| Commits | 8 |
| Estimated LOC added | ~500 |

---

## Conclusion

The experiment suggests three changes worth baking into skills:

1. **Lean context** - Don't make subagents read full plan files; extract what they need
2. **Self-reflection** - Ask implementers to critique their own work before handoff
3. **Separate ruthless review** - Fresh eyes with explicit "find problems" mandate

The self-reflection + review combo caught bugs that would have shipped otherwise. The overhead is worth it for any non-trivial implementation.
