# Architecture decision records

ADRs capture **meaningful architectural decisions that are not already dictated by
the technical specification**.

## When to write one

Write an ADR when a decision:

- is expensive to reverse;
- constrains future implementation options;
- is not answerable by reading the specification;
- would otherwise live only in chat history.

## When NOT to write one

- The specification already decides it (cite the spec instead).
- Trivial implementation choices (naming, file layout inside a package, formatting).
- Anything a work item's `notes` field can carry.

Do not generate speculative ADRs.

## Template

```markdown
# ADR-NNN: Title

## Status

Proposed / Accepted / Superseded by ADR-XXX

## Context

What is the situation, and what forces matter?

## Decision

What was decided, in one or two sentences, then in detail.

## Consequences

What becomes easier, what becomes harder, what is now excluded.
```

Number ADRs sequentially from `001`. File them as `docs/adr/adr-NNN-title.md`.
