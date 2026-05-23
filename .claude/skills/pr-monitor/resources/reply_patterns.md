# Reply patterns + structurally-stable assertions

Patterns that make review iteration go faster and reduce follow-up nits.

## Triggering CodeRabbit auto-resolution

CR's auto-resolve detector fires only on an **inline thread reply** that matches both:

1. Starts with `@coderabbitai` (or contains it early; `@coderabbitai` is the detector trigger)
2. Contains a commit SHA reference, ideally as a markdown link `[SHA](commit-url)`

### Template

```markdown
@coderabbitai Addressed in [SHA](https://github.com/{owner}/{repo}/pull/{n}/commits/{SHA}).

<one or two sentences on what you changed and any deviations from CR's suggested diff>

Closing out.
```

### What does NOT trigger auto-resolve

- Commit messages mentioning "Address CodeRabbit review" — CR doesn't watch commit messages
- A top-level PR comment that lists addressed items — fires for the human maintainer but not the bot
- An inline reply that mentions a SHA but not `@coderabbitai`

### Use `reply_to_thread.sh`

```bash
.claude/skills/pr-monitor/scripts/reply_to_thread.sh \
  aallan/vera-bench 73 3289486639 bfbfae2 \
  "Applied your suggested diff verbatim. Two new tests pin the new behavior."
```

## When CR's suggested diff is wrong

CR sometimes suggests diffs with subtle errors (e.g., wrong literal values, mismatched conventions). Don't apply blindly. The right pattern when correcting:

1. Apply the **intent** of the finding (which is usually correct)
2. Deviate from the literal diff where needed
3. **Explain the deviation** in the inline reply — this builds trust and helps the reviewer

### Real example from `aallan/vera-bench` PR #70

CR suggested testing AILANG boolean normalisation with mocked stdout `"True\nFalse"` (capital T, Python `repr()` style). But AILANG actually outputs `"true"/"false"` (lowercase, matching Aver). Applying the diff verbatim would have produced a test that asserted the wrong contract.

The reply that closed the thread cleanly:

> @coderabbitai Addressed in [d6769c4]. Sharp catch — applied the intent verbatim, with one deliberate adjustment: your mocked stdout used capital `"True\nFalse"` (Python's bool repr), but that's not what AILANG outputs. AILANG (like Aver) prints **lowercase** `true`/`false`, so capital-`True` against expected `"true"` would actually fail the matcher. Using lowercase to pin the real contract.

CR's response: "that's a sharp catch — the uppercase True/False in my suggested mock was Python's repr, not AILANG's output, and would have been testing the wrong contract entirely."

## Structurally-stable assertions

The strongest tests assert on **structural properties**, not message text. CR will flag string-substring assertions as brittle.

### Examples

```python
# BRITTLE: couples test to message wording
assert "invalid" not in result.output.lower()
assert "Worker crash" in line

# STABLE: structural / semantic
assert result.exit_code != 2  # Click parse error has a fixed exit code
crash_row = next(row for row in rows if row["problem_id"] == "VB-X-2")
```

```python
# BRITTLE: asserts implementation (which class got imported)
with patch("concurrent.futures.ThreadPoolExecutor", side_effect=AssertionError):
    run_with_parallel_one()

# STABLE: asserts behavior (no worker thread was spawned)
calls_on_main = [
    t for t in calling_threads if t is threading.main_thread()
]
assert len(calls_on_main) == len(problems)
```

```python
# BRITTLE: couples to log format
captured = capsys.readouterr()
assert "Running benchmark" in captured.out

# STABLE: observable side effect
assert output_path.exists()
assert json.loads(output_path.read_text().splitlines()[0])["problem_id"] == "VB-X-0"
```

The principle: a refactor that doesn't change behavior shouldn't break a behavior-asserting test. If the test fires on string content, it's actually testing the string content, not the behavior.

## Closeout comments for changes-requested reviews

When the reviewer asked for several things, structure the reply as a checklist mirroring their numbering. Cite the addressing commit SHA inline next to each item.

### Template

```markdown
@reviewer — addressed in [SHA]. Going item-by-item:

### Important (all in)

- [x] **I1**  — <what the ask was> — <how addressed>. <Any deviation explained>.
- [x] **I2** — ...
- [x] **I3** — ...
- [x] **I4** — ...
- [x] **I5** — ...

### Suggestions

- [x] **S2** — ...
- [x] **S4** — ...

- [ ] **S1** — deferred: <reason>.
- [ ] **S3** — kept as-is: <reason>.

<closing summary, 1-2 sentences>
```

Why this format works:
- The reviewer can match each `[x]` against the line numbers they listed
- Items that DIDN'T land have explicit `[ ]` with a reason — no silent skips
- The reviewer scrolls through 30 seconds, sees everything mapped, can approve
