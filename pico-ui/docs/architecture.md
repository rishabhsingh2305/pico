# Architecture

## Overview

pico-ui is the browser client for the picoservice Go API. It is a React single
page application built with Vite. The current features are account signup,
login, email verification, an organization dashboard, organizations, projects,
and project debits and credits.

```text
browser
  -> React signup / login / verification / dashboard / organization / project / expense pages
  -> requestJSON (fetch, credentials: include)
  -> VITE_API_BASE_URL (default http://localhost:8080) or the production API origin
  -> picoservice Lambda Function URL
  -> POST /users, POST /auth/login, POST /auth/logout, GET /auth/session,
     POST /auth/verify-email, GET/POST /orgs, GET/POST /projects,
     GET/POST/DELETE /expenses
  -> JSON response
```

The backend sends a verification email as part of `POST /users`. The email link
opens the UI at `/verify-email#token=...`; the UI reads the fragment and calls
the API. After verification the user signs in at `/login`, which sets an
HttpOnly JWT cookie. `AuthProvider` restores the session from `/auth/session` on
load.

## Structure

```text
src/
├── app/
│   └── router.tsx                   # Route table
├── components/
│   ├── app-shell.tsx                # Sidebar shell for signed-in screens
│   ├── page-header.tsx              # Shared page title and actions
│   ├── brand.tsx                    # Shared Pico wordmark
│   ├── date-range-picker.tsx        # Preset/custom date range picker
│   ├── tag-input.tsx                # Shared badge-style tag entry
│   ├── theme-provider.tsx           # Existing light/dark/system provider
│   └── ui/                          # Generated shadcn primitives
├── features/
│   ├── api-docs/
│   │   ├── content.ts               # Curated endpoint reference data
│   │   ├── types.ts                 # Endpoint/section types
│   │   ├── components/              # Cards, code blocks, copy, navigation
│   │   └── pages/
│   │       └── api-docs-page.tsx
│   ├── auth/
│   │   ├── api.ts                   # Signup, login, logout, session, verification
│   │   ├── auth-provider.tsx        # Session context and protected-route guard
│   │   ├── error-message.ts         # Safe signup and login error mapping
│   │   ├── session.ts               # Session shape and invalid-session detection
│   │   ├── types.ts                 # SignUpInput, LoginInput, User, SessionResponse
│   │   ├── validation.ts            # Pure signup and login validation
│   │   ├── components/
│   │   │   ├── auth-shell.tsx
│   │   │   ├── login-form.tsx
│   │   │   ├── sign-up-form.tsx
│   │   │   └── sign-up-success.tsx
│   │   └── pages/
│   │       ├── login-page.tsx
│   │       ├── sign-up-page.tsx
│   │       └── verify-email-page.tsx
│   ├── dashboard/
│   │   ├── data.ts                  # Currency grouping and flow-bar helpers
│   │   ├── components/
│   │   │   ├── dashboard-summary.tsx
│   │   │   ├── project-flow.tsx
│   │   │   ├── recent-activity.tsx
│   │   │   └── section-heading.tsx
│   │   └── pages/
│   │       └── dashboard-page.tsx
│   ├── design-system/
│   │   └── pages/
│   │       └── style-tile-page.tsx   # Public palette/type/pattern reference
│   ├── expenses/
│   │   ├── api.ts                   # GET/POST/DELETE expenses
│   │   ├── error-message.ts         # Safe expense error mapping
│   │   ├── filters.ts               # Tag/date filter parsing
│   │   ├── money.ts                 # Decimal <-> minor units, formatting
│   │   ├── types.ts                 # Expense, ExpenseType, ExpensePage, CreateExpenseInput
│   │   ├── validation.ts            # Pure expense validation
│   │   └── components/
│   │       ├── create-expense-form.tsx
│   │       ├── expense-filter-form.tsx
│   │       └── expense-list.tsx
│   ├── orgs/
│   │   ├── api.ts                   # GET/POST/PATCH orgs, GET org detail
│   │   ├── error-message.ts         # Safe org error mapping
│   │   ├── types.ts                 # Org, CreateOrgInput
│   │   ├── validation.ts            # Pure org name/tag validation
│   │   ├── components/
│   │   │   ├── create-org-form.tsx
│   │   │   ├── edit-org-tags-dialog.tsx
│   │   │   ├── org-card.tsx
│   │   │   └── org-filter-form.tsx
│   │   └── pages/
│   │       ├── orgs-page.tsx
│   │       ├── org-detail-page.tsx
│   │       └── project-detail-page.tsx
│   └── projects/
│       ├── api.ts                   # GET/POST/PATCH projects
│       ├── error-message.ts         # Safe project error mapping
│       ├── types.ts                 # Project, CreateProjectInput
│       ├── validation.ts            # Pure project name/tag validation
│       └── components/
│           ├── create-project-form.tsx
│           ├── edit-project-tags-dialog.tsx
│           ├── project-card.tsx
│           └── project-filter-form.tsx
├── hooks/                           # Shared hooks and page title
├── lib/
│   ├── api-client.ts                # JSON request wrapper and ApiError
│   ├── currencies.ts                # Supported currency codes
│   ├── date-range.ts                # Date range presets and labels
│   ├── env.ts                       # Public API base URL
│   ├── tags.ts                      # Shared tag normalization/validation
│   └── text.ts                      # Byte-length helper
├── test/
│   └── setup.ts                     # Testing Library setup
├── App.tsx                          # Renders the router
└── main.tsx                         # Providers and BrowserRouter
```

## Signup flow

1. Unauthenticated users land on `/login`; `/signup` is linked from there.
2. `SignUpPage` renders the form.
3. `validateSignUp` checks required fields, email shape, and byte limits.
4. `signUp` posts `{name,email,password}` to `/users`.
5. A `201` response switches the page to the success state with the entered
   email.
6. Errors stay on the form and are shown through an accessible alert.

The form is disabled while the request is in flight. The success state offers a
"Use a different email" action that returns to an empty form.

## Login flow

1. `LoginPage` renders the email and password form.
2. `validateLogin` checks the email shape and required password.
3. `AuthProvider.login` posts `{email,password}` to `/auth/login`.
4. The backend sets an HttpOnly `pico_auth` JWT cookie; the UI stores the
   returned user in memory and navigates to `/orgs`.
5. `401` maps to "Email or password is incorrect." and `403` maps to "Verify your
   email before signing in."
6. On reload, `AuthProvider` calls `GET /auth/session` to restore the user. A
   `401` clears the session and protected routes redirect to `/login`.

Tokens are never read by JavaScript or stored in `localStorage`. Signing out
calls `/auth/logout` and clears the cookie.

## Email verification flow

1. The backend sends a link to `{PUBLIC_BASE_URL}/verify-email#token=...`.
2. The UI reads `token` from the URL fragment.
3. `VerifyEmailPage` calls `POST /auth/verify-email` with `{"token":"..."}`.
4. On success, the page shows a "Continue to sign in" action.
5. The page shows loading, verified, expired/invalid, missing-token, or network
   states.
6. Network failures offer a retry action; the backend verification is
   idempotent.

The fragment is not sent to the static host or its access logs. The token is
sent to the API only in the POST body. Verification does not create a session;
the user signs in afterwards.

## Authenticated workspace

Protected routes render inside `RequireSession`, which shows a loading state
while `/auth/session` resolves and redirects to `/login` when there is no
session.

- `/` redirects to `/orgs` when authenticated, otherwise `/login`.
- `GET /orgs?tags=...&limit=100` loads the organization list.
- `POST /orgs` creates an organization and its owner membership.
- `PATCH /orgs/{orgID}?orgId=...` updates the owner's organization name or tags.
- A `401` or a stale `404 user not found` clears the session and redirects to
  `/login`.
- New organizations are prepended when no tag filter is active; otherwise the
  filtered list reloads.
- The workspace sidebar shows the signed-in email and offers sign out.

The legacy `userId` query parameter is still sent while the backend supports the
temporary fallback; the auth cookie is the authoritative identity.

## Workspace shell

Signed-in screens share `AppShell`: a persistent sidebar from the `lg` breakpoint,
a compact mobile top bar with a sheet navigator, active navigation state, and a
footer that shows the signed-in user with a sign-out action. `PageHeader`
provides the shared eyebrow, title, description, and action layout. Auth screens
use `AuthShell` with a split brand panel, while `/api-docs` keeps its public
header so it works without a session.

## Organization dashboard

`/orgs/:orgId/dashboard` is the organization dashboard. It is opened from the
organization's project list and is scoped to that organization; there is no
global dashboard or organization switcher.

- KPIs show the net total, credit, and debit for the selected currency. The net
  uses the oversized `text-metric` display; credit is moss (`chart-2`) and debit
  is vermilion (`primary`).
- Monetary totals are grouped by project currency. When an organization uses
  more than one currency, currency tabs switch the KPI and flow views. Amounts
  from different currencies are never added together.
- The project flow chart is a pure CSS ledger: one row per project in the active
  currency, with debit extending left, credit extending right, and the project
  net on the right. Bars scale to the largest credit or debit in the group.
- Recent activity lists the latest six expenses for the organization. Each row
  shows its own currency code because the list may mix currencies.
- The dashboard loads `GET /orgs/{orgID}`, `GET /projects`, one
  `GET /expenses?projectId=...&limit=1` per project for totals, and one
  organization-wide `GET /expenses?limit=6` for recent activity. It never derives
  totals from a returned page.
- Empty states cover an organization with no projects and no recent activity.
  Network and access errors offer a retry.

## Organization and project workspace

`/orgs/:orgId` is the project list. `/orgs/:orgId/projects/:projectId` is the
project expense page.

- Projects are listed on the organization page; there is no separate Projects
  section. The New project action opens a dialog containing the create form.
- Organization cards show their tags. Clicking a tag toggles it in the compact
  “Filter tags (any)” toolbar and reloads the organization list from the server.
- Organization tag filtering uses any-match semantics: selecting team and
  finance returns organizations with either tag. This matches project tag
  filtering and differs from expense tag filtering, which requires every tag.
- Owners can edit organization tags from the card. The list reloads after an
  edit because the organization may no longer match the active filter.
- Creating an organization while a filter is active reloads the filtered list;
  without a filter, the new organization is prepended.
- The filtered empty state is “No organizations match these tags.”
- The organization detail page shows the organization's tags below its title.
- The project list links to the organization dashboard with a Dashboard action
  next to New project.
- The organization page shows project names, currencies, and tags, with no
  expense information.
- Each project has a currency selected at creation; it is shown next to the
  project name.
- Project tags can be edited from the project card. The list reloads after an
  edit because the project may no longer match the active filter.
- Clicking a project name or arrow opens its expense page.
- The project page loads the organization, project, and that project's debits and
  credits.
- The project identity (eyebrow, name, organization, tags) and the New expense
  action sit in a compact toolbar above the net hero. On large screens the net
  hero spans the full width, the filter toolbar follows, and the ledger fills the
  remaining viewport height: only the expense list scrolls, so the net total and
  filters stay visible. On smaller screens the page uses normal document
  scrolling.
- The net summary is the page's largest type. It uses the container-relative
  `text-metric-fluid` size so the amount can grow to fill the full-width hero
  without overflowing, and a single formula line shows the credit minus debit
  values that produce it. An ink chip labels the figure, and the accent rule is
  moss for a positive net and vermilion for a negative one.
- The create form exists only inside a modal and is not rendered until the New
  expense button is pressed.
- Each expense is a debit (money out) or credit (money in). Amounts are always
  positive; the type supplies the direction. The create form defaults to debit
  and sends the selected type.
- Expenses inherit the project currency; the expense form has no currency field.
- Creating or deleting an expense reloads the current filter and page so the
  summary stays correct. A successful create closes the modal.
- If there are no projects, the organization page prompts for one.
- If a project has no expenses, its page shows an empty state.
- If filters match nothing, the list shows a filtered empty state and the
  summary shows zero.

### Project tags and filtering

- Project tags are created in the project form and edited in a modal from the
  project card.
- Tags use the shared badge input. Comma or Enter commits a draft, blur commits
  a pending draft, duplicates are ignored, and badges are removable.
- The project filter is a compact toolbar labeled “Filter tags (any)” with a tag
  input and Clear. It has no Apply button because tags apply immediately.
- Adding or removing a filter tag, or clicking a tag badge on a project card,
  reloads the project list from the server.
- Project tag filtering uses **any-match** semantics: selecting food and travel
  returns projects with food, travel, or both. This intentionally differs from
  expense tag filtering, which requires every tag.
- Filtered empty state: “No projects match these tags.”
- Clearing the filter reloads the unfiltered project list.

### Expense filters, totals, and pagination

- Filter controls render as a compact toolbar. The tag field is labeled “Filter
  tags (all)”; it shares a row with one date range picker, Apply, and Clear on
  desktop. The toolbar stacks into compact rows on mobile and has no card
  heading or helper copy.
- Tags use a badge input shared by the filter toolbar and the create form.
  Typing a comma or pressing Enter commits the draft into a removable badge;
  blur also commits a pending draft. Every filter badge must match.
- Duplicate tags are ignored, and Backspace on an empty draft removes the last
  badge.
- The date picker is a single button that opens presets (Today, Last 7 days,
  Last 30 days, This month, Last month) and a custom start/end calendar. Presets
  include today; “This month” ends today rather than at month end.
- The selected dates are inclusive local calendar dates. The client converts
  the from date to the start of day and the to date to the end of day, then sends
  RFC 3339 timestamps to the API.
- The client rejects an inverted range before making a request; the backend
  validates it again.
- `totalCount`, `totalDebit`, `totalCredit`, and `netTotal` describe every
  expense matching the filters. They come from the API and are never derived
  from the current page. `netTotal` is `totalCredit - totalDebit`.
- The list shows 20 expenses per page. Previous and Next change only `offset`;
  the server-computed totals stay the same.
- Applying filters resets `offset` to zero. Clear resets tags and dates and
  reloads the first page.
- Totals are in minor units and are formatted with the project currency. The
  list rows show a debit or credit badge and a signed amount; the large animated
  total shows the net, with credit and debit figures beside it.

### Money conversion

Expense amounts are stored by the backend as integer minor units. The form
accepts a decimal amount and converts it using the project currency exponent:

```text
12.99 USD -> 1299
100 JPY   -> 100
1.234 KWD -> 1234
```

The conversion uses string parsing, not floating point. Display uses
`Intl.NumberFormat`. Amounts remain positive after conversion; debit/credit is a
separate type, and net totals subtract debits from credits.

## API documentation page

`/api-docs` is public and does not require a session. The page is a
curated guide rather than generated Swagger output.

- Content lives in typed data in `src/features/api-docs/content.ts`.
- Sections cover health, authentication, users, email verification,
  organizations, memberships, projects, expenses, and common error behavior.
- Each endpoint shows its method, path, access level, parameters, request/response
  examples, errors, and important notes.
- Search filters endpoints by method, path, title, section, parameters, errors,
  and notes. It hides empty sections and shows a filtered empty state.
- Copy buttons use `navigator.clipboard` for paths, request bodies, and response
  examples.
- The layout uses a sticky desktop sidebar and a collapsible mobile section
  navigator.
- The page links back to `/orgs` when authenticated, otherwise `/login`.
- The app header and auth shell link to the guide.

The backend README and architecture document remain the canonical API contract.
When an endpoint, request shape, response shape, or rule changes, update
`src/features/api-docs/content.ts` in the same change.

## API contract

### `POST /auth/login`

Request:

```json
{
  "email": "ada@example.com",
  "password": "correct horse"
}
```

Success: `200` with `{"user":{...},"expiresAt":"..."}` and an HttpOnly
`pico_auth` JWT cookie. Invalid credentials return `401`; unverified accounts
return `403`. Login never returns the token in the response body.

### `POST /auth/logout`

Success: `204`. Clears the auth cookie. Tokens are stateless, so an already
issued token remains valid until it expires.

### `GET /auth/session`

Success: `200` with `{"user":{...},"expiresAt":"..."}`. Missing, invalid, or
expired cookies return `401`.

### `POST /users`

Request:

```json
{
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "password": "correct horse"
}
```

Success: `201` with a user object containing `id`, `name`, `email`,
`emailVerified`, `createdAt`, and `updatedAt`.

The backend validates and normalizes the input again. Client validation only
improves feedback; it is not a security boundary.

### `POST /auth/verify-email`

Request:

```json
{
  "token": "..."
}
```

Success: `200` with `{"status":"verified","userId":"...","email":"..."}`.
Invalid, expired, or mismatched tokens return `400` with a generic message.

### `GET /orgs?userId=...&tags=food,travel&limit=100`

Success: `200` with
`{"items":[{"id","name","userId","tags","createdAt","updatedAt"}]}`. Only
organizations where the user has a membership are returned. `tags` uses
any-match semantics: an organization is returned when it has at least one of the
requested tags. Absent or empty tags returns all organizations. Organizations
without a stored tags field read as an empty list.

### `POST /orgs?userId=...`

Request:

```json
{
  "name": "Acme Inc.",
  "tags": ["team", "finance"]
}
```

Success: `201` with the created organization. The backend also creates the
owner membership. Names are required and capped at 120 bytes. Tags are optional,
trimmed, deduplicated, sorted, and capped at 20 items of 32 bytes each.

### `GET /orgs/{orgID}?orgId=...&userId=...`

Success: `200` with the organization, including `tags`. The user must be a
member. The path ID is canonical; `orgId` is repeated in the query for
`RequireMember`.

### `PATCH /orgs/{orgID}?orgId=...&userId=...`

Request with either or both fields:

```json
{
  "name": "Acme Labs",
  "tags": ["team", "finance"]
}
```

Success: `200` with the updated organization. At least one field is required.
Only the owner may update. `tags: []` clears all organization tags.

### `GET /projects?orgId=...&userId=...&tags=food,travel&limit=100`

Success: `200` with `{"items":[...]}`. Projects belong to the organization.
`tags` uses any-match semantics: a project is returned when it has at least one
of the requested tags. Absent or empty tags returns all projects.

### `GET /projects/{projectID}?orgId=...&userId=...`

Success: `200` with the project. The user must be a member of the organization.

### `POST /projects?orgId=...&userId=...`

Request:

```json
{
  "name": "General",
  "currency": "USD",
  "tags": ["travel", "food"]
}
```

Success: `201` with the created project. Names are required and capped at 120
bytes. Currency is a three-letter ISO code and is immutable after creation.
Tags are trimmed, deduplicated, sorted, and capped at 20 items of 32 bytes each.

### `PATCH /projects/{projectID}?orgId=...&userId=...`

Request with either or both fields:

```json
{
  "name": "Renamed",
  "tags": ["new"]
}
```

Success: `200` with the updated project. At least one field is required.
`tags: []` clears all project tags.

### `GET /expenses?orgId=...&userId=...&projectId=...&tags=...&from=...&to=...&limit=20&offset=0`

Success: `200` with:

```json
{
  "items": [],
  "totalCount": 42,
  "totalDebit": 100000,
  "totalCredit": 123456,
  "netTotal": 23456
}
```

Expenses are scoped to the organization. `projectId` limits them to one project.
`tags=food,travel` requires every tag to match. `tag` remains available as a
single-tag fallback. `from` and `to` are inclusive RFC 3339 bounds. `limit` and
`offset` paginate `items` only; `totalCount`, `totalDebit`, and `totalCredit`
cover every matching expense, and `netTotal` is `totalCredit - totalDebit`.
Totals are in minor units and, on the project page, use that project's currency.

### `POST /expenses?orgId=...&userId=...`

Request:

```json
{
  "projectId": "...",
  "type": "debit",
  "amount": 1299,
  "description": "Lunch",
  "tags": ["food"],
  "spentAt": "2026-01-02T12:30:00Z"
}
```

`type` is `debit` for money out or `credit` for money in. It defaults to debit
when omitted. `amount` is a positive integer in minor units; `type` carries the
direction. The project must belong to the organization. Currency is inherited
from the project. Records without a type are read and aggregated as debit.

### `PATCH /expenses/{expenseID}?orgId=...&userId=...`

Request with any subset of `projectId`, `type`, `amount`, `description`, `tags`,
and `spentAt`. At least one field is required. `type`, when supplied, must be
`debit` or `credit`. Currency is inherited from the project and cannot be
changed.

### `DELETE /expenses/{expenseID}?orgId=...&userId=...`

Success: `204`.

## Error mapping

| Condition         | UI message                                                       |
| ----------------- | ---------------------------------------------------------------- |
| `400`             | The server's validation message                                  |
| `409`             | An account with this email already exists.                       |
| `502`             | We couldn't send the verification email. Please try again.       |
| Network failure   | Could not reach the server. Check your connection and try again. |
| Other `4xx`/`5xx` | Something went wrong. Please try again.                          |

Internal server details are never displayed.

Verification errors use their own page states:

- `400` — invalid or expired link
- missing fragment token — incomplete link
- network or unexpected failure — retry action
- success — verified message

Organization errors:

- `400` — server validation message
- `403` — permission message
- `401` — expired or invalid session; clear it and redirect to login
- `404` — stale `userId` fallback; clear the session and redirect to login
- network or other failures — retry or generic message

## Configuration

| Variable                | Scope      | Purpose                                     |
| ----------------------- | ---------- | ------------------------------------------- |
| `VITE_API_BASE_URL`     | build time | API origin, default `http://localhost:8080` |
| `VITE_API_PROXY_TARGET` | dev server | Optional proxy target; unused by default    |

- The browser calls the API directly. The Go API allows `PUBLIC_BASE_URL`
  through `middleware.CORS` with credentials, so the HttpOnly `pico_auth` cookie
  is sent.
- Production should use `SameSite=None; Secure` for a cross-site cookie. Local
  development uses `SameSite=Lax` because localhost ports are same-site.
- `VITE_*` values are public and are embedded in the bundle.
- Copy `.env.example` to `.env.local` for local overrides. `.env` files are
  ignored; `.env.example` is committed.

## Routing

`src/app/router.tsx` is the single route table:

- `/signup` — signup page
- `/login` — sign-in page
- `/verify-email` — reads the fragment token and verifies it
- `/api-docs` — public, searchable API reference
- `/design` — public design-system style tile
- `/orgs` — organization list and creation
- `/orgs/:orgId` — project list for an organization
- `/orgs/:orgId/dashboard` — organization dashboard
- `/orgs/:orgId/projects/:projectId` — project expenses
- `/` — redirects to `/orgs` when authenticated, otherwise `/login`
- `*` — redirects through `/`

Protected routes render inside `RequireSession`.

`BrowserRouter` lives in `main.tsx` so tests can use `MemoryRouter`. A static
host must serve `index.html` for unknown paths.

## Decisions

### Vite, React, and TypeScript

The existing scaffold already provides this stack. It keeps the app small and
fast without a server framework.

### shadcn/ui and Tailwind

The repository already contains shadcn primitives and theme tokens. Reusing them
keeps the auth page consistent with future screens and avoids a second design
system.

### Editorial Brutalist visual system

The interface uses warm paper surfaces, ink borders, vermilion primary actions,
moss credit values, and acid-lime navigation. Bricolage Grotesque carries
display headings and money figures; Inter carries UI and body text. The custom
`text-display`, `text-title`, `text-metric`, `text-metric-fluid`,
`text-heading`, and `text-label` utilities in `src/index.css` define the scale,
so hierarchy is applied through tokens rather than one-off font sizes.
`text-metric-fluid` is container-relative and is used for the project ledger net
so it scales to its column. `/design` is the public living style tile for the
palette, type scale, and core patterns.

### React Router now

A router is included because signup, login, and verification are separate
screens. Route registration stays in one file.

### Plain React state instead of a form library

The signup form has three fields and one request. A pure validation function and
local state are enough. Add a form library only when multiple forms repeat the
same behavior.

### Fetch wrapper instead of a data library

`requestJSON` provides typed JSON parsing and a single `ApiError` shape. Caching
and retries are not needed for one request. Add a data library when list/detail
screens need shared caching.

### Strong generated-code lint exception

`src/components/ui/**` and `src/hooks/use-mobile.ts` are shadcn-generated and
disable `react-hooks/set-state-in-effect`. Application code keeps the strict
rule.

### HttpOnly JWT cookie session

Login posts credentials to `/auth/login`, which sets an HttpOnly `pico_auth`
JWT cookie. The browser sends it automatically because the UI and API allow each
other through CORS with credentials. `AuthProvider` keeps the user in memory and
restores it with `GET /auth/session`; no token is readable by JavaScript or
stored in `localStorage`. Tokens are stateless, so sign-out clears the cookie
without server-side revocation.

### Project-owned currency

Currency belongs to the project, not to each expense. The project form selects
it at creation, the project list and project page display it, and the expense
form converts decimal input using that currency. The backend copies the project
currency into each expense when it is created.

### Debit/credit direction on expenses

An expense amount is always positive. A `type` field records whether the money
left (`debit`) or arrived (`credit`), so direction is not encoded in the sign of
a minor-unit amount. Project totals are sums of both directions:
`netTotal = totalCredit - totalDebit`. Legacy records without a type are treated
as debit on read and in Mongo aggregation, so no migration is required. The UI
shows a badge and signed amount per row and keeps the large net total with a
credit/debit breakdown.

### Project and organization tags filter on the server

Project and organization filtering is server-side so results are not limited by
whichever records happened to be loaded by a previous request. Both use
any-match semantics while expense tags use all-match semantics: projects and
organizations are containers with several organizing labels, whereas expense
tags usually narrow a spending view. The UI sends selected tags as `tags=...`
and reloads only the affected list. Organization tags are owner-editable from
the card; the organization list is assembled from membership IDs, so filtering
applies within the fetched membership page.

### Shared tag input and validation

Tag entry is identical across organizations, projects, and expenses, so the
badge input lives in `src/components/tag-input.tsx` and normalization limits live
in `src/lib/tags.ts`. This avoids a feature importing another feature's internals
and keeps the backend's 20-tag/32-byte contract in one UI module.

### Date range presets on the existing calendar

The expense date filter uses one picker built from the installed
`react-day-picker` calendar and popover instead of two native date inputs or a
new date-range dependency. Presets are pure helpers in `src/lib/date-range.ts`
so their inclusive boundaries are unit tested. The picker writes the same
`yyyy-MM-dd` form values as before, which keeps the RFC 3339 conversion and the
backend contract unchanged.

### Curated API guide instead of Swagger

The API is small enough that a hand-curated guide is clearer than generated
Swagger output. It can explain identity caveats, membership scenarios, tag
semantics, and money conversion in plain language. Content is typed and kept in
one data module, while the backend documents remain the canonical contract. The
tradeoff is that contract changes must update both places; the docs rule records
that obligation.

### Server-side filtered totals

The expense list response carries debit, credit, and net totals computed by the
backend over the same filters without pagination. The UI never sums the current
page, so paging does not change the displayed total and filters are the single
source of truth. A client-side total would be wrong as soon as there is more
than one page.

### Currency-grouped dashboard totals

An organization can hold projects in different currencies, and minor units are
not comparable across them. The dashboard therefore loads each project's totals
with `limit=1` and groups them by currency before summing. A single
organization-wide expense total would be numerically wrong for mixed-currency
organizations. The tradeoff is one totals request per project; at the current
scale that is acceptable, and a future backend summary endpoint can replace it
without changing the screen.

### Viewport-contained project ledger

On desktop the project expense page is a fixed-height workspace: the project
identity and New expense action form a compact toolbar, the net hero spans the
full width, filters stay in place, and the expense list scrolls inside the
remaining space. This makes the net total the dominant figure while keeping it
visible during a long ledger scan. The net uses a container-relative type size so
it can grow into the full-width hero without overflowing, and the layout falls
back to normal page scrolling below `lg` or when the fixed content cannot fit.

### Expense creation in a modal

The project page keeps the expense list visible and exposes creation through a
single New expense button. The form lives in a dialog so the page is not
dominated by an empty form, and because creation is occasional rather than the
primary reading task. The form defaults to debit and can record a credit. The
dialog has a title, description, and Cancel action; successful creation closes
it and refreshes the current page and totals.

## Known debt

- Access tokens are stateless and cannot be revoked before they expire; signing
  out clears the cookie only.
- The legacy `userId` query parameter is still sent alongside the auth cookie
  until the backend removes the temporary fallback.
- No resend UI; an expired link has no recovery path in the UI yet.
- No organization settings or membership screens.
- The editorial visual system currently covers the shell, page headers, org
  list/detail, filter toolbars, and the project ledger. Auth, dialogs, and API
  docs still use the older component-level styling.
- No standalone project screens beyond the project expense page.
- Project tag editing is available, but project rename/delete is not in the UI.
- No expense edit; list/create/delete only.
- No debit/credit filter in the expense toolbar; both directions are shown
  together.
- Server-side project tag filtering still uses a 100-project page with no
  pagination UI.
- Server-side organization tag filtering applies within the 100-membership page
  because the list is assembled from membership IDs. A merged membership/org
  query would be needed at larger counts.
- The dashboard issues one totals request per project. A backend summary
  endpoint grouped by currency and project would remove this N+1 pattern.
- Date filters use the browser's local timezone; there is no timezone selector.
- No global state or server-state cache.
- No deployment configuration or static-host SPA rewrite.
- `VITE_API_BASE_URL` is fixed at build time, so each environment needs its own
  build.
- No end-to-end or visual regression tests.

## Extending the UI

1. Add a feature folder with types, API, validation, components, and pages.
2. Keep API calls in the feature's `api.ts`.
3. Register routes in `src/app/router.tsx`.
4. Add colocated Vitest + Testing Library tests.
5. Update this document, `docs/engineering-standards.md`, and `README.md` when the
   contract or setup changes.
