# Plan: Correct the empty-expense call to action

## Context

Repository: `pico-ui` — React 19, TypeScript, Vite, Tailwind CSS 4, shadcn/ui.

Audited surface: project expense page at `/orgs/:orgId/projects/:projectId`.

Current commit: `8a63152`. The working tree already contains uncommitted feature work; do not revert unrelated changes.

## Problem

The empty expense state tells users to add the first expense “above,” but the create-expense form exists only inside a modal opened by the **New expense** button.

Evidence:

- `src/features/expenses/components/expense-list.tsx:58` renders `Add the first expense for this project above.`
- `docs/architecture.md:160-162` states that the create form exists only inside the modal.
- `src/features/orgs/pages/project-detail-page.tsx:344-368` renders the form inside the **New expense** dialog.
- No inline expense form exists in the page.

## Required change

In `src/features/expenses/components/expense-list.tsx`, change only the empty-state secondary copy from:

```text
Add the first expense for this project above.
```

to:

```text
Use New expense to add the first expense.
```

Keep:

- the `No expenses yet` heading
- the filtered empty state copy
- the icon
- the card layout
- all spacing and classes

## Tests

In `src/features/orgs/pages/project-detail-page.test.tsx`, extend the existing empty-expense test to assert:

```ts
expect(
  screen.getByText("Use New expense to add the first expense.")
).toBeInTheDocument()
```

Confirm the filtered empty state still shows its existing copy.

## Docs

No contract or architecture change is required. This aligns the copy with the already documented modal behavior.

## Acceptance criteria

- Empty expense state references **New expense**, not a form above.
- Filtered empty state remains unchanged.
- `corepack pnpm format:check`
- `corepack pnpm typecheck`
- `corepack pnpm lint`
- `corepack pnpm test`
- `corepack pnpm build`

all pass.
