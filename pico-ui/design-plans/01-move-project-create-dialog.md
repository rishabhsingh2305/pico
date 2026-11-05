# Plan: Move project creation into a Dialog

## Context

Repository: `pico-ui` — React 19, TypeScript, Vite, Tailwind CSS 4, shadcn/ui.

Audited surface: signed-in organization/project workspace at `/orgs/:orgId`.

Current commit: `8a63152`. The working tree already contains uncommitted feature work; do not revert unrelated changes.

## Problem

The project creation form is rendered inline inside a `Card`, while the documented UI pattern requires occasional create/edit forms to use the shadcn `Dialog` with a title, description, and Cancel action.

Evidence:

- `docs/engineering-standards.md:96-98` requires occasional create/edit forms in `Dialog`.
- `src/features/orgs/pages/org-detail-page.tsx:254-263` renders `CreateProjectForm` inline.
- `src/features/projects/components/create-project-form.tsx:95-97` wraps the form in `Card`.
- `src/features/orgs/pages/project-detail-page.tsx:344-368` and `src/features/projects/components/edit-project-tags-dialog.tsx` are existing modal exemplars using `Dialog`, `DialogTrigger`, `DialogContent`, `DialogHeader`, `DialogTitle`, `DialogDescription`, and `triggerId`.

## Required change

Render project creation only inside a `Dialog`, triggered by the existing **New project** action.

### `src/features/orgs/pages/org-detail-page.tsx`

1. Import `useId` and the required `Dialog` primitives:
   - `Dialog`
   - `DialogContent`
   - `DialogDescription`
   - `DialogHeader`
   - `DialogTitle`
   - `DialogTrigger`
2. Replace `showProjectForm` state with `isProjectDialogOpen`.
3. Remove the `autoCreateForm`, `showCreateForm`, and related inline-rendering logic.
4. Render a single `Dialog` in the Projects section:
   - Trigger: existing `New project` button with `PlusIcon`.
   - Use the controlled `open`/`onOpenChange` plus `triggerId` pattern from `project-detail-page.tsx`.
   - Keep the trigger visible when `projectsStatus === "ready"`, including when there are no projects.
   - `DialogContent` should use `className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg"`.
   - Header:
     - `DialogTitle`: `Create project`
     - `DialogDescription`: explain that projects own the currency and tags used by their expenses and filters.
   - Body: `CreateProjectForm`.
5. On successful create:
   - close the dialog
   - call the existing project reload logic
6. On cancel:
   - close the dialog
7. Update the empty-project state copy from “Create a project above…” to refer to the **New project** action.

Keep the current project loading, filtering, tag-edit, and empty-filtered states unchanged.

### `src/features/projects/components/create-project-form.tsx`

1. Remove the `Card`/`CardContent` wrapper and related imports.
2. Return the form directly, matching the structure of `src/features/expenses/components/create-expense-form.tsx`.
3. Make the Cancel action required for the modal context if that keeps the API simpler; the form is currently used only on the organization page.
4. Preserve:
   - labels and ids
   - tag input and validation
   - loading/disabled behavior
   - error alert
   - request payload

## Tests

Update `src/features/orgs/pages/org-detail-page.test.tsx`:

- “prompts for a project when none exist”:
  - project form fields are **not** initially present
  - **New project** button is present
  - clicking it opens a dialog named `Create project`
  - project name/currency/tags fields become available
- “creates a project with tags and adds it to the list”:
  - click **New project** before filling the form
  - submit through the dialog
  - assert the dialog closes
  - keep the existing POST body and list-refresh assertions
- Add a cancel-path assertion:
  - open the dialog
  - click `Cancel`
  - assert the dialog closes and no request is made
- Existing filtering, tag editing, loading, error, and stale-session tests must continue to pass.

## Docs

Update places that currently describe inline project creation:

- `AGENTS.md:13` — project creation opens in a modal.
- `docs/architecture.md:149` — project creation is controlled by the **New project** dialog.
- `README.md:167` — clarify that project creation and tag editing happen on the organization page; creation uses a modal.
- `docs/engineering-standards.md` — no change; this plan aligns the implementation with the existing rule.

## Acceptance criteria

- No inline project form remains on the organization page.
- **New project** opens a titled modal with description and Cancel.
- Successful creation closes the modal and reloads the current filtered list.
- Empty-project state points to **New project**.
- Project creation still sends `{ name, currency, tags }`.
- `corepack pnpm format:check`
- `corepack pnpm typecheck`
- `corepack pnpm lint`
- `corepack pnpm test`
- `corepack pnpm build`

all pass.
