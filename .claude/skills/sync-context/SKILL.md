---
name: sync-context
description: Reconcile CLAUDE.md, the agent definitions, and the skills against the code as it actually is. Use after any change to events/, outbox/, subscribers/internal/handler/, platform/amqp/, a go.mod, or a migration — and whenever a doc names a symbol that no longer compiles.
---

# Sync the context files to the code

Agents read `CLAUDE.md` and the skills **before** they read the code. A stale doc does not merely fail to help — an agent will implement the design it read rather than the one that exists. This skill closes that gap.

The files under review:

```
CLAUDE.md
.claude/agents/*.md
.claude/skills/*/SKILL.md
```

## 1. Detect

Start from the diff, not from the docs. What did you actually change?

```bash
git diff --name-only HEAD -- events outbox subscribers platform api '*.mod' '*.sql'
```

For every symbol you removed, renamed, or resignatured, grep the docs for it. A doc naming a symbol that does not compile is stale, without exception:

```bash
# every exported identifier the docs mention, checked against the tree
rg -o '`[A-Z][A-Za-z0-9_]*\.[A-Za-z0-9_]+`' CLAUDE.md .claude/ | sort -u
```

Then the standing check — symbols known to be gone. Empty output is the pass condition:

```bash
rg -n -f .claude/retired-symbols.txt CLAUDE.md .claude/agents .claude/skills
```

`.claude/retired-symbols.txt` holds one regex per retired identifier. The patterns live in a data file, not inline in a doc, because a check written inside the text it searches always matches itself and can never pass.

**Add a line to that file whenever you retire a symbol.** It is the regression test for the docs, and it is only as good as the last person who fed it.

## 2. Reconcile

Rewrite the doc to describe what is now true.

**Delete the superseded guidance. Do not leave it beside its successor.** Two descriptions of one mechanism are worse than one wrong description, because a reader cannot tell which is current. If the old design is worth remembering, one sentence saying what replaced it and why is enough — that is context, not instruction.

Keep the *reasons*. A doc that says "dispatch on the declared payload type" teaches nothing. One that says "dispatch on the declared payload type, never `reflect.TypeOf`, because the row outlives the binary that wrote it" survives the next refactor, because it tells the reader which forces are in play.

## 3. Verify

Nothing goes in a doc that has not been checked.

- **Every Go snippet compiles** against the current tree. Types, function names, argument counts, argument order.
- **Every SQL snippet runs** against the current schema. Column names, status values.
- **Every path exists.** `ls` it.
- **Every command works.** Run it.

A snippet that guards a real operational procedure does not belong in prose alone — put it in a test. `outbox/tests/integration/recovery_test.go` runs the documented recovery `UPDATE` verbatim, so the runbook cannot rot silently.

If you cannot verify a claim, delete it rather than shipping it. An unverified doc is a confident lie.

## 4. Report

Name each file you changed and the code change that forced it. If you found a doc describing a design that never existed, or found code with no doc that needed one, say so — those are the interesting findings.

## Checklist

- [ ] `git diff` reviewed; every renamed/removed symbol grepped across `CLAUDE.md` and `.claude/`
- [ ] Standing stale-symbol grep returns empty
- [ ] Superseded guidance deleted, not left alongside its replacement
- [ ] Every Go snippet in every touched doc compiles against the tree
- [ ] Every SQL snippet matches the current schema
- [ ] Reasons preserved, not just rules
- [ ] Operational SQL that matters is covered by a test, not only prose
- [ ] Report names each doc and the change that forced it
