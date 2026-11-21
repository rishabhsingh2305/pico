# Expenses

React + TypeScript + Vite UI for expense management. Authentication is provided
by the separate `picoservice` API.

## Development

```bash
cp .env.example .env
pnpm install
pnpm dev
```

The dev server proxies `/api` to `http://localhost:8080` (`picoservice`). Change
the target with `VITE_API_PROXY_TARGET`.

## Authentication

- Signup posts to `/users` and triggers a 6-digit verification email.
- The code is verified with `/auth/verify-email/otp`.
- Login posts to `/auth/login`; the JWT is stored in `localStorage` and sent as
  `Authorization: Bearer`.
- The session is restored from `/auth/session` on page load.
- After login the app silently finds the internal org tagged `expenses`, or
  creates it on first use. The org is never shown in the UI; only `orgId` is
  passed to project and expense API calls.

## Categories

- The main page lists categories, backed by the `/projects` endpoints.
- Creating a category asks for a name only. New categories use `INR` and no tags
  by default; the currency is not exposed in the UI yet.
- Opening a category shows its expenses, net/spent/received totals from
  `/expenses/summary`, and an **Add expense** dialog.
- The totals collapse into a compact strip when scrolling down the expense list
  and expand again when scrolling back toward the top.
- Expenses can be marked **Spent** (debit) or **Received** (credit), with an
  amount, optional description, optional tags, and date.
- Tags are added with Enter, space, or comma, shown as chips in the expense
  list, and capped at 20 tags of 32 characters each.
- The expense list can be filtered by date preset or custom range and by tags;
  all selected tags must match.
- Home category blocks show this month's spend.

## Sharing

- Users who own the org or carry `share.admin` can create read-only links for a
  category (backed by a project). Category sharing is available from the
  category list and from the category detail header.
- Links use 7, 30, or 365-day expiry presets and can be revoked from the same
  dialog. Public badges appear on shared categories.
- A link opens `/shared` with an opaque token in the URL fragment
  (`#share=...`). The page keeps the token in the URL, stores a copy in
  `sessionStorage`, and sends `Authorization: Share` on the existing read
  routes. Refreshing or copying the URL keeps the link usable.
- The shared summary shows net balance, credit (money in), and debit (money out)
  per currency. Rows from categories that share a currency are combined, and
  negative balances keep their sign instead of being displayed as absolute
  values.
- Single-expense links show the expense's credit or debit amount in the same
  summary layout.
- Multi-expense links load 20 expenses at a time. Scrolling near the bottom, or
  pressing **Load more**, appends the next page. The summary stays based on the
  full filtered set rather than the visible page.
- `/shared` sits outside the session guard and the personal Expenses org
  provider, so recipients do not need an account.

## Production

Set `VITE_API_BASE_URL` to the deployed `picoservice` URL. The UI origin must
also be listed in the service's `CORS_ALLOWED_ORIGINS`.
