# DECISIONS.md — Architecture Decision Records

> **Purpose**: Log significant technical decisions and their rationale.

## Template

```markdown
## [DECISION-XXX] Title

**Date**: YYYY-MM-DD
**Status**: Proposed | Accepted | Deprecated | Superseded

### Context
What is the issue we're facing?

### Decision
What have we decided to do?

### Rationale
Why did we make this decision?

### Consequences
What are the trade-offs?

### Alternatives Considered
What other options were evaluated?
```

---

## Decisions

### Phase 1 Decisions

**Date:** 2026-03-19

### Scope
- **Repository**: Using `github.com/harsh-sreehari/sortd`.
- **Documentation**: Root-level technical docs (`SPEC.md`, `README.md`, `ARCHITECTURE.md`, `GSD-STYLE.md`, `PROJECT_RULES.md`) moved to `docs/` to keep the root clean for the Go project.

### Approach
- **Go Module**: Initialized as `github.com/harsh-sreehari/sortd`.
- **Storage**: Using raw SQL with `modernc.org/sqlite` instead of an ORM.
- **Scaffolding**: Directly implementing functional modules rather than an empty skeleton.

### Constraints
- **User Identity**: Operations should use the identity of `harsh-sreehari`.
### Phase 8 Decisions

**Date:** 2026-03-20

### Scope
- **Renaming**: Context-Rich renaming (Option B). AI will generate descriptive names based on content analysis (e.g. `Scan_001.jpg` -> `College_AlgorithmAnalysis_Quiz.jpg`).
- **Learning**: Scrapped the `tutor` command in favor of **Teaching Mode**. Corrected moves in `sortd review` will now feed the `affinities` table to bias future LLM decisions for similar file patterns.
- **Pruning**: Added safety checks to ensure root drives/folders are accessible before pruning missing records.

### Approach
- **Rename Logic**: Always suggest to user first. If the file exists, the AI should suggest a *different* name instead of just appending `_1`.
- **Affinities**: Used as "In-Context Examples" or "Reinforcement Prompting" in Tier 3.

### Constraints
- **Concurrency**: Re-runs `sortd index` cleanup to ensure real-time accuracy after manual corrections.

---

*Last updated: <!-- date -->*
