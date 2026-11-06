# pico-ui

The browser client for picoservice. Built with React 19, TypeScript, Vite,
Tailwind CSS 4, shadcn/ui, and React Router.

Current scope: signup, login, email verification, an organization dashboard,
organization list/create, project list/create, and project expenses.
Organizations carry tags and can be filtered by any matching tag. Each
organization’s project list links to its dashboard, which summarizes that
organization with currency-grouped net, credit, and debit totals, a project flow
chart, and recent activity. Opening an organization shows its projects, which
can carry tags and be filtered by any matching tag. Opening a project shows that
project's debits and credits, with tag/date filters, a server-computed net total,
and 20-item pagination.

The UI uses an editorial “paper and ink” design system: warm parchment surfaces,
vermilion and moss data accents, acid-lime navigation, Bricolage Grotesque
display type, and Inter for UI text. Open `http://localhost:5173/design` for the
living style tile with the palette, type scale, and core patterns.

## Requirements

- Node.js 22+
- Corepack and pnpm 12
- The picoservice backend running locally on `:8080`

## Setup

```bash
corepack enable
corepack pnpm install
cp .env.example .env.local
```

The default `.env.example` points the UI directly at the Go API at
`http://localhost:8080`. The API allows `http://localhost:5173` through CORS and
accepts the UI origin, so bearer requests work without a proxy.

## Commands

| Command                      | Purpose                               |
| ---------------------------- | ------------------------------------- |
| `corepack pnpm dev`          | Start Vite at `http://localhost:5173` |
| `corepack pnpm test`         | Run Vitest once                       |
| `corepack pnpm test:watch`   | Run Vitest in watch mode              |
| `corepack pnpm typecheck`    | Type-check the app                    |
| `corepack pnpm lint`         | Run ESLint                            |
| `corepack pnpm format`       | Format with Prettier                  |
| `corepack pnpm format:check` | Verify formatting                     |
| `corepack pnpm build`        | Type-check and build to `dist/`       |
| `corepack pnpm preview`      | Preview the production build          |

## Local development

Start the backend:

```bash
cd ../picoservice
cp sample.env .env   # first time only
# Edit .env. PUBLIC_BASE_URL must be http://localhost:5173 for the UI link.
make run
```

Start the UI in another terminal:

```bash
corepack pnpm dev
```

Open `http://localhost:5173`. Signup posts directly to `POST /users`. The
backend sends a verification email. Clicking the link opens
`/verify-email#token=...` in the UI, which calls the verification API. After
verification, sign in at `/login`; the backend returns an access token, which
the UI stores in `localStorage` and sends as `Authorization: Bearer`. The UI
restores the session from `/auth/session` on reload. Then
continue to `/orgs` to create and list organizations. Organizations accept tags
at creation, show them on their cards,
and can be filtered with a tag toolbar; owners can edit tags from the card. Open
an organization to see and create projects; projects have editable tags and can
be filtered server-side by any matching tag. The organization’s project list
also links to its dashboard, which groups totals by project currency and shows
project flow and recent entries. Open a project to manage its expenses. The
project page records each entry as a debit (money out) or credit (money in),
filters by comma-separated tags (all must match) and a date range picker with
presets or custom start/end dates, shows the filtered net total with credit and
debit breakdowns, and paginates the expense list without changing that total. On
desktop the ledger fills the viewport and only the expense list scrolls, keeping
the net total visible.

Open `http://localhost:5173/api-docs` for the public, in-app API reference. It
includes searchable endpoint cards, request/response examples, and copy buttons.

## Configuration

| Variable                | Scope      | Description                                 |
| ----------------------- | ---------- | ------------------------------------------- |
| `VITE_API_BASE_URL`     | build time | API origin, default `http://localhost:8080` |
| `VITE_API_PROXY_TARGET` | dev server | Optional proxy target; unused by default    |

`VITE_*` values are public and embedded in the browser bundle. Never put
secrets in this project. The browser calls the API directly and sends the access
token in the `Authorization` header. The API allows the UI origin through CORS.

## Routes

| Path                               | Screen                     |
| ---------------------------------- | -------------------------- |
| `/signup`                          | Signup page                |
| `/login`                           | Sign-in page               |
| `/verify-email`                    | Email verification         |
| `/api-docs`                        | Public API documentation   |
| `/orgs`                            | Organization list + create |
| `/orgs/:orgId`                     | Project list + create      |
| `/orgs/:orgId/dashboard`           | Organization dashboard     |
| `/orgs/:orgId/projects/:projectId` | Project expenses           |
| `/`                                | Session-aware redirect     |
| `*`                                | Redirect through `/`       |

## API integration

- `src/features/api-docs/content.ts` is the curated in-app API reference shown
  at `/api-docs`. It documents every backend endpoint, identity rules, examples,
  errors, pagination, and tag-filter semantics.
- `src/lib/api-client.ts` parses JSON and throws `ApiError` with the HTTP
  status.
- `src/features/auth/api.ts` calls `POST /users` and
  `POST /auth/verify-email`.
- `src/features/orgs/api.ts` calls `GET/POST /orgs`, `GET /orgs/{id}`, and
  `PATCH /orgs/{id}` for owner tag edits. `GET /orgs` sends `tags` for
  any-match filtering. Organization creation includes tags. Tags are normalized
  with the shared 20-tag/32-byte limits.
- `src/features/projects/api.ts` calls `GET/POST /projects`,
  `GET /projects/{id}`, and `PATCH /projects/{id}` for tag edits. Project
  creation includes the currency and tags. The project list sends `tags` to the
  server, where any matching tag includes the project.
- `src/features/expenses/api.ts` calls `GET/POST /expenses` and
  `DELETE /expenses/{id}`. Expenses inherit currency from their project and carry
  a `type` of `debit` (money out) or `credit` (money in). Amounts stay positive;
  the type supplies the direction.
- `src/features/dashboard/pages/dashboard-page.tsx` composes the org, project,
  and expense APIs. It is opened from the organization’s project list. It loads
  one `limit=1` expense summary per project, groups those totals by currency, and
  loads an organization-wide `limit=6` page for recent activity. `projectId` is
  omitted from the org-wide request.
- `GET /expenses` returns
  `{"items","totalCount","totalDebit","totalCredit","netTotal"}`. `items` is the
  current page; the totals cover every expense matching the filters and are not
  affected by `limit` or `offset`. `netTotal` is `totalCredit - totalDebit`.
  Amounts are in minor units and are formatted with the project currency.
- The dashboard never adds minor units across currencies. It shows currency tabs
  when an organization has projects in more than one currency, and each recent
  activity row carries its own currency code.
- The organization page shows project names, currencies, and tags. Project tags
  are clickable and toggle the active server-side filter; the filter toolbar
  also accepts manual tag badges. Tags can be edited from each project card.
- Project expenses are opened from a project card.
- The project page loads its expenses and provides a New expense button,
  delete controls, tag/date filters, a filtered net total with credit and debit
  breakdowns, and 20-item pagination. Expense creation opens in a modal; the
  form is not shown inline and defaults to a debit with an option to record a
  credit. Filters use a compact toolbar, and tag entry creates removable badges
  on comma or Enter. Applying filters resets to the first page; creating or
  deleting reloads the current page and totals.
- `src/features/auth/error-message.ts` maps signup and login errors to safe
  messages.
- Success responses switch to the check-your-email state.
- The verification page reads `token` from the URL fragment and posts it to the
  API, then sends the user to `/login`.
- `/login` posts credentials to `/auth/login`; the backend returns an access
  token. `AuthProvider` stores it in `localStorage` and restores the session from
  `/auth/session`.
- Protected routes render inside `RequireSession`; the workspace sends `orgId`
  and the legacy `userId` query parameter while the backend migrates fully to
  bearer authentication.
- Expense amounts are entered in major units and converted to minor units using
  the currency exponent.

## Tests and checks

```bash
corepack pnpm format:check
corepack pnpm typecheck
corepack pnpm lint
corepack pnpm test
corepack pnpm build
```

Tests use Vitest and React Testing Library. `fetch` is mocked; the suite does not
touch the network.

## Project docs

- [`AGENTS.md`](AGENTS.md) — agent operating rules and definition of done
- [`docs/architecture.md`](docs/architecture.md) — structure, flow, API, decisions
- [`docs/engineering-standards.md`](docs/engineering-standards.md) — clean code,
  React, accessibility, testing

## Known limitations

- Access tokens are stateless and cannot be revoked before they expire; signing
  out clears the stored token only. The token is in `localStorage`, so it is
  readable by JavaScript if XSS occurs.
- No resend UI yet.
- No organization settings or membership screens.
- No standalone project management screens; project creation opens in a dialog
  on the organization page, tag editing is on project cards, and project expenses
  are on the project page.
- Project rename/delete is not implemented in the UI.
- No expense edit; create, list, and delete only.
- Project lists are capped at 100 with no pagination UI.
- Organization lists are capped at 100 and organization tag filtering applies
  within that page. A merged membership/org query would be needed at larger
  counts.
- The dashboard issues one totals request per project. A backend summary
  endpoint grouped by currency and project would remove this N+1 pattern.
- Existing projects without a currency or tags read as USD and an empty tag
  list. Existing expenses without a type read as debit.
- No deployment or static-host SPA rewrite configuration yet.
