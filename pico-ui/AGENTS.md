# pico-ui agent instructions

## Project snapshot

- React 19, TypeScript, Vite 8, Tailwind CSS 4, and shadcn/ui (`base-nova`).
- React Router 8 for client-side routing.
- Editorial “paper and ink” visual system: Bricolage Grotesque display type,
  Inter UI text, warm parchment surfaces, vermilion primary, moss credit, and
  acid-lime navigation. `/design` is the public living style tile.
- pnpm 12 via Corepack. Do not use npm or yarn.
- Vitest 5 and React Testing Library for UI tests.
- The backend is the Go service in `../picoservice`.
- Current scope is signup, login, email verification, an organization dashboard,
  organization list/create, project list/create, and project debits and credits.
  An organization shows project names, currencies, and tags; opening a project
  shows that project's debits and credits. Projects are created in a dialog from
  the organization page, own the currency used by their expenses, and own tags
  used to filter the project list. Organizations also carry tags, shown on their
  cards, editable by owners, and filterable with any-match semantics.
- `/orgs/:orgId/dashboard` is the organization dashboard, opened from the
  project list. It shows currency-grouped net, credit, and debit totals, a
  project flow chart, and the latest expense entries. Monetary totals are never
  summed across currencies; the dashboard groups by currency and offers a
  currency switch.
- Project expenses are debits (money out) or credits (money in). Amounts stay
  positive; the type carries direction. Existing records without a type are
  treated as debit. They support comma-separated tags (all must match), an
  inclusive local date range, a server-computed net total with credit and debit
  breakdowns, and 20-item pagination. Pagination must never change the totals;
  creating or deleting reloads the current filter and page.
- The project page shows only a New expense button. The create form appears in
  a modal, defaults to debit, can record a credit, closes on success or Cancel,
  and is not rendered inline.
- Projects have tags that can be added at creation, edited from the project
  card, and filtered server-side with any-match semantics. Clicking a project
  tag toggles it in the compact filter toolbar.
- `/api-docs` is a public, searchable, in-app API reference covering every
  backend endpoint. Content lives in typed data under `src/features/api-docs`.
- Verification emails contain a link and a 6-digit code. The UI verifies via
  the link only; `POST /auth/verify-email/otp` is API-only for now. Resend
  rotates the code and returns `Retry-After: 60`.
- Organization settings, memberships, project rename/delete, and expense edit are
  not implemented yet.
- Login posts to `/auth/login`; the backend returns an access token.
  `AuthProvider` stores it in `localStorage`, sends it as
  `Authorization: Bearer`, restores the session from `/auth/session` on load,
  and exposes login, logout, and the current user.

## Read first

- `docs/architecture.md` — structure, request flow, API integration, decisions.
- `docs/engineering-standards.md` — clean code, React, accessibility, testing.
- `../picoservice/README.md` — backend API contract and local setup.

## Commands

| Command                      | Purpose                             |
| ---------------------------- | ----------------------------------- |
| `corepack pnpm dev`          | Start the Vite dev server           |
| `corepack pnpm test`         | Run Vitest once                     |
| `corepack pnpm test:watch`   | Run Vitest in watch mode            |
| `corepack pnpm typecheck`    | Type-check the app                  |
| `corepack pnpm lint`         | Run ESLint                          |
| `corepack pnpm format`       | Format with Prettier                |
| `corepack pnpm format:check` | Verify formatting                   |
| `corepack pnpm build`        | Type-check and build for production |
| `corepack pnpm preview`      | Preview the production build        |

## Structure

```text
src/
├── app/                    # Router and app-level composition
├── components/
│   ├── app-shell.tsx         # Sidebar workspace shell for signed-in screens
│   ├── page-header.tsx       # Shared page title and actions
│   ├── brand.tsx             # Shared Pico wordmark
│   ├── date-range-picker.tsx # Preset/custom date range picker
│   ├── tag-input.tsx         # Shared badge-style tag entry
│   ├── ui/                   # Generated shadcn primitives
│   └── theme-provider.tsx    # Existing theme provider
├── features/
│   ├── api-docs/           # Public searchable API reference
│   ├── auth/               # Signup, login, verification, session
│   ├── dashboard/          # Workspace totals, project flow, recent activity
│   ├── design-system/      # Public palette/type/pattern style tile
│   ├── expenses/           # Expenses, debit/credit, filters, totals, money
│   ├── orgs/               # Organization list/detail, tags, filtering
│   └── projects/           # Inline project list/create, tags, filters
├── hooks/                  # Shared hooks and page title
├── lib/                    # Framework-free utilities and API client
└── test/                   # Vitest setup
```

Boundaries:

- `lib/` must not import features or components.
- Feature code may import `lib/`, `components/ui/`, and its own files.
- Feature modules must not import another feature's internals. Extract shared
  code only after a second real use.
- `components/ui/` contains generated shadcn primitives. Add components with
  the shadcn CLI; do not hand-edit them except for generated-code fixes.
- Pages live in `src/features/<feature>/pages`; route registration lives in
  `src/app/router.tsx`.
- Tests are colocated as `*.test.ts` or `*.test.tsx` next to the code they cover.

## Non-negotiable rules

- **Clean code:** KISS and YAGNI first. Apply DRY after the third real
  repetition. Keep components and functions small and explicit.
- **TypeScript:** no `any`. Use explicit API types and type-only imports.
  `verbatimModuleSyntax` and `erasableSyntaxOnly` are enabled.
- **API calls:** only feature `api.ts` modules may call `requestJSON`. Components
  never call `fetch` directly. Use `ApiError` and map statuses to safe messages.
- **List totals:** expense `totalCount`, `totalDebit`, `totalCredit`, and
  `netTotal` come from the API and cover the full filtered set. `netTotal` is
  `totalCredit - totalDebit`. Never derive them from the current page. Applying
  filters resets pagination; changing pages leaves the totals unchanged.
  Dashboard summaries must group minor-unit totals by project currency and never
  add amounts across currencies.
- **Tag filters:** project and organization tags filter server-side with
  any-match semantics; expense tags require all tags to match. Keep this
  difference explicit in the UI and docs.
- **State:** local React state first. Do not add a global store or data-fetching
  library until there is a repeated need. The authenticated session lives in
  `AuthProvider`, restores from `/auth/session`, and stores the access token in
  `localStorage`. Never store passwords there.
- **Forms:** controlled fields with a small pure validation function. Mirror
  backend constraints (name 120 bytes, email 254 bytes, password 8–72 bytes).
  Projects own a currency; expense amounts are positive decimal values converted
  to integer minor units using that project currency, and each expense carries a
  debit or credit type. Organizations and projects share the tag input and the
  20-tag/32-byte limits. Expense filters use removable tag badges where all must
  match, and one date range picker with presets or a custom range; its inclusive
  local dates convert to RFC 3339 day bounds. The server remains authoritative.
- **Accessibility:** every input has a label. Use `aria-invalid`,
  `aria-describedby`, keyboard submit, visible focus, and an alert role for
  errors.
- **Styling:** Tailwind utilities and shadcn tokens. Use `cn`; do not add
  hardcoded colors outside the theme. Support light and dark mode. Use
  `font-display` with `text-display`, `text-title`, `text-metric`,
  `text-metric-fluid` (container-relative project net), `text-heading`, or
  `text-label` instead of one-off display sizes. Credit is `chart-2`, debit is
  `primary`. Keep `/design` in sync when tokens change.
- **Routing:** register routes in `src/app/router.tsx`. `/` redirects to
  `/orgs` when authenticated, otherwise `/login`. Protected routes render inside
  `RequireSession`.
- **Environment:** only `VITE_*` variables belong in the browser bundle. They are
  public. Never put secrets in the UI. Keep `.env.example` current.
- **Tests:** every behavior change needs a test. Mock `fetch`; never call a real
  network or external service. Cover success, validation, and error paths.
- **Dependencies:** prefer the existing stack. Justify new dependencies, verify
  maintenance, and update the lockfile with pnpm.
- **Docs:** update `README.md`, `docs/architecture.md`,
  `docs/engineering-standards.md`, and this file when routes, API contracts,
  architecture, or conventions change. API contract changes must also update
  `src/features/api-docs/content.ts` so the public guide stays accurate.
- **Generated files:** do not edit `node_modules`, `dist`, or lockfiles by hand.
  Do not edit `src/components/ui/` except to fix generated-code issues.

## Adding a page or feature

1. Add types and the API function under `src/features/<feature>/`.
2. Add pure validation and error mapping where needed.
3. Build small components and a page. Keep JSX free of business rules.
4. Register the route in `src/app/router.tsx`.
5. Add colocated Vitest + Testing Library tests.
6. Update the docs and `README.md` if the contract or setup changed.
7. Run `format:check`, `typecheck`, `lint`, `test`, and `build`.

## Definition of done

- [ ] Behavior is covered by tests.
- [ ] `corepack pnpm typecheck` passes.
- [ ] `corepack pnpm lint` passes.
- [ ] `corepack pnpm test` passes.
- [ ] `corepack pnpm build` passes.
- [ ] `corepack pnpm format:check` passes.
- [ ] Accessibility and mobile layout were checked.
- [ ] No secrets, `.env` files, `dist/`, or unrelated generated changes.
- [ ] Docs are updated when the contract or architecture changes.
