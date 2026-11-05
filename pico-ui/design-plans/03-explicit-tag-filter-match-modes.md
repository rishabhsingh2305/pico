# Plan: Make tag-filter match modes explicit

## Context

Repository: `pico-ui` — React 19, TypeScript, Vite, Tailwind CSS 4, shadcn/ui.

Audited surfaces:

- Project filter on `/orgs/:orgId`
- Expense filter on `/orgs/:orgId/projects/:projectId`

Current commit: `8a63152`. The working tree already contains uncommitted feature work; do not revert unrelated changes.

## Problem

The UI uses different tag-filter semantics but does not state them:

- Project tags use **any-match**.
- Expense tags use **all-match**.

The repository explicitly requires this difference to be visible in the UI.

Evidence:

- `AGENTS.md:96-97` requires project any-match and expense all-match semantics to remain explicit in the UI and docs.
- `src/features/projects/components/project-filter-form.tsx:28` labels the project filter only `Filter tags`.
- `src/features/expenses/components/expense-filter-form.tsx:68` labels the expense filter only `Tags`.
- `docs/architecture.md:181` and `:493` document the difference, but the filter surfaces do not expose it.

## Required change

Keep the compact filter layout and add the match mode to the visible field labels.

### Project filter

File: `src/features/projects/components/project-filter-form.tsx`

Change:

```tsx
<FieldLabel htmlFor={tagsId}>Filter tags</FieldLabel>
```

to:

```tsx
<FieldLabel htmlFor={tagsId}>Filter tags (any)</FieldLabel>
```

### Expense filter

File: `src/features/expenses/components/expense-filter-form.tsx`

Change:

```tsx
<FieldLabel htmlFor={tagsId}>Tags</FieldLabel>
```

to:

```tsx
<FieldLabel htmlFor={tagsId}>Filter tags (all)</FieldLabel>
```

Do not add helper paragraphs or change filter behavior. The parenthetical is intentionally short so the filter remains a single compact toolbar row on desktop.

## Tests

Update label queries:

- `src/features/orgs/pages/org-detail-page.test.tsx`
  - Replace `getByLabelText("Filter tags")` with `getByLabelText("Filter tags (any)")`.
- `src/features/orgs/pages/project-detail-page.test.tsx`
  - In filter-specific tests, replace `getByLabelText("Tags")` with `getByLabelText("Filter tags (all)")`.
  - Do not change queries scoped to the create-expense dialog, where the field remains `Tags`.

Add assertions that both labels exist in their respective filter surfaces if the existing label queries do not already cover them.

## Docs

Update the UI architecture filter section to mention that visible labels carry the match mode:

- Project filter: `Filter tags (any)`
- Expense filter: `Filter tags (all)`

No backend or API contract changes.

## Acceptance criteria

- Project filter visibly states any-match.
- Expense filter visibly states all-match.
- Create/edit tag fields still use the label `Tags`.
- Compact toolbar layout is preserved.
- `corepack pnpm format:check`
- `corepack pnpm typecheck`
- `corepack pnpm lint`
- `corepack pnpm test`
- `corepack pnpm build`

all pass.
