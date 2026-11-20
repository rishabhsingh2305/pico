# picoservice

A small Go REST API running as a single AWS Lambda function behind a public
Lambda Function URL. Routing is handled by
[chi](https://github.com/go-chi/chi), with a small adapter that translates
Function URL events into standard `net/http` requests. MongoDB Atlas stores
users, organizations, memberships, projects, expenses, and short links.

## Endpoints

All request and response bodies are JSON, except the public short-link
redirect, which returns `302` with no body. List responses accept `limit` and
`offset` query parameters (default `20`, maximum `100`). Expense and link lists
use `{"items":[...],"nextOffset":...}`; expense totals come from
`GET /expenses/summary`. Read endpoints also support `HEAD`, except the email
verification action, which is `POST` only.

When pico-ui is running, open `http://localhost:5173/api-docs` for a searchable
in-app API guide with endpoint examples and copy buttons. This README remains the
canonical API contract.

### Health

| Method | Path     | Response                                                              |
| ------ | -------- | --------------------------------------------------------------------- |
| GET    | `/ping`  | `{"message":"pong"}` (`200`)                                          |
| HEAD   | `/ping`  | empty body (`200`)                                                    |
| GET    | `/ready` | `{"status":"ok"}` (`200`) or `{"error":"database unavailable"}` (`503`) |
| HEAD   | `/ready` | empty body (`200` or `503`)                                           |

### Authentication

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/auth/login` | Sign in with `{"email","password"}`; returns `{"accessToken","refreshToken","tokenType","user","expiresAt","refreshExpiresAt","orgs"}` and sets `pico_auth` and `pico_refresh` cookie fallbacks |
| `POST` | `/auth/refresh` | Exchange `{"refreshToken":"..."}` or the `pico_refresh` cookie for a new access and refresh token |
| `POST` | `/auth/logout` | Clear both auth cookies (`204`); the client clears its stored tokens |
| `GET` | `/auth/session` | Return the current user, token expiry, and `orgs` role map from `Authorization: Bearer` or the cookie, or `401` |

Access tokens are HS256 JWTs valid for 30 minutes and carry `sub`, `iss`,
`aud=picoservice-api`, `type=access`, and `orgs`. The `orgs` map is capped at 20
organizations to keep the header small. Refresh tokens are valid for
30 days with `type=refresh`. Refreshing re-queries the user's memberships, so a
role change takes effect within one access-token lifetime. Login requires a
verified email: wrong credentials return `401 invalid email or password`, and
unverified accounts return `403 email not verified`. Tokens are stateless;
logout and refresh rotation do not revoke already-issued tokens.

### Users

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/users` | Register with `{"name","email","password"}`; sends a verification email |
| `GET` | `/users/me` | Get the authenticated user |
| `PATCH` | `/users/me` | Change `name` and/or `email`. Changing the email resets verification |
| `POST` | `/users/me/password` | Change password with `currentPassword` and `newPassword` |

Protected routes use `Authorization: Bearer` (or the `pico_auth` cookie
fallback). The actor is always the token `sub`; `?userId=` is not accepted as
identity. There is no global user directory.

Passwords are bcrypt-hashed. Password hashes are never returned. There is no
`DELETE /users` yet. User responses include `emailVerified`.

### Email verification

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/auth/verify-email` | Verify with `{"token":"..."}` from the UI link |
| `POST` | `/auth/verify-email/otp` | Verify with `{"email":"...","code":"123456"}` |
| `POST` | `/auth/verify-email/resend` | Resend with `{"email":"..."}`; returns `202 {"status":"sent","retryAfterSeconds":60}` with `Retry-After: 60`, throttled to once per 60 seconds |

Verification links contain a signed JWT valid for 24 hours and return
`{"status":"verified","userId":"...","email":"..."}`. Signup creates the user
as unverified. The same email also contains a 6-digit code valid for 10 minutes
with at most 5 failed attempts. Changing the
email marks the account unverified and sends a new link and code. Resend
responses are deliberately generic so they do not reveal whether an email is
registered.

### Organizations

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/orgs` | Create `{"name","tags"}`; the actor becomes owner and a membership is created |
| `GET` | `/orgs?tags=food,travel` | List orgs the actor belongs to; filters: `tags` (any match) |
| `GET` | `/orgs/{orgID}?orgId={orgID}` | Get an org; any role |
| `PATCH` | `/orgs/{orgID}?orgId={orgID}` | Partial update of `name` and/or `tags`; requires `org.admin` or the owner wildcard |

Organization tags are trimmed, deduplicated, sorted, and capped at 20 items with
a 32-byte limit per tag. `tags=food,travel` returns organizations matching
**any** of the requested tags. `tags: []` on PATCH clears all tags. Existing
organizations without a tags field read as an empty list.

Path-scoped org routes repeat `orgId` in the query and it must match the path.
`RequireOrgAccess` reads the tenant from `?orgId=`, the feature roles from the
token's `orgs` map, and the required unit permission from `routes.yml`.

There is no `DELETE /orgs`.

### Memberships

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/orgs/{orgID}/members?orgId={orgID}` | Add `{"userId":"...","roles":{"expense":"editor"}}`; requires `member.admin` or the owner wildcard. `roles` defaults to the legacy editor map |
| `GET` | `/orgs/{orgID}/members?orgId={orgID}` | List members; owner and `member.viewer`/`member.admin` |
| `PATCH` | `/orgs/{orgID}/members/{memberUserID}?orgId={orgID}` | Replace `{"roles":{"project":"viewer","expense":"editor"}}`; requires `member.admin`; the owner cannot change |
| `DELETE` | `/orgs/{orgID}/members/{memberUserID}?orgId={orgID}` | Remove a member; requires `member.admin`; the owner cannot be removed |

Membership roles are feature-scoped: `{"org":"viewer","member":"admin","project":"editor","expense":"viewer","link":"viewer"}`. The org's `userId` is the `owner`, which is implicit, has every feature role, is never stored as a membership role, and is written as `"*"` in access tokens. The member response uses `owner: true` for the owner and `roles` for everyone else.

`member.admin` is a high-trust role: it can add and remove members and assign their feature roles, so grant it only to trusted administrators. The owner cannot be added, removed, or have their roles changed.

### Sharing

Share links are opaque credentials. The raw token is returned once and stored
only as a SHA-256 hash. The recipient can read the shared resource through the
existing API routes.

| Method | Path | Permission | Description |
| --- | --- | --- | --- |
| `POST` | `/orgs/{orgID}/shares?orgId={orgID}` | `shares.create` | Create a link. Returns the raw token once |
| `GET` | `/orgs/{orgID}/shares?orgId={orgID}` | `shares.list` | List links without tokens |
| `GET` | `/orgs/{orgID}/shares/status?orgId={orgID}&resourceType=expense&resourceIds=...` | `shares.status` | Bulk “is this resource shared?” lookup |
| `GET` | `/orgs/{orgID}/credentials?orgId={orgID}&kind=share&...` | `shares.credentials` | Filter links by kind, resource type, IDs, and active state |
| `DELETE` | `/orgs/{orgID}/shares/{shareID}?orgId={orgID}` | `shares.revoke` | Revoke a link |

Create body:

```json
{
  "resourceType": "project",
  "resourceId": "project-1",
  "name": "Client review",
  "expiresAt": "2026-12-31T23:59:59Z"
}
```

`resourceType` is `org`, `project`, or `expense`. Scopes are derived server-side.
`expiresAt` defaults to 30 days and is capped at 365 days. The create response
returns the raw `token` and a `sharePath` such as
`/shared?orgId=org-1&projectId=project-1&resourceId=project-1&resourceType=project#share=pico_share_...`.
Each client turns that path into a link on its own origin.

The recipient sends:

```http
GET /expenses/expense-42?orgId=org-1
Authorization: Share <opaque-token>
```

Share scopes:

| Share type | Read routes |
| --- | --- |
| `org` | org, projects, expenses, summary |
| `project` | project, project expenses, summary |
| `expense` | one expense, its project, the org |

A project share forces the project filter. An expense share can only read the
expense ID it was created for. Share responses omit `userId`.

The `share` feature governs management:

```yaml
share:
  viewer: [shares.list, shares.status, shares.credentials]
  admin: [shares.list, shares.status, shares.credentials, shares.create, shares.revoke]
```

The owner wildcard covers everything.

### Short links

Short links are public redirects managed by an organization. Anyone with the
URL can open it without authentication. Creating, listing, editing, disabling,
and deleting require a `link.*` role.

| Method | Path | Permission | Description |
| --- | --- | --- | --- |
| `POST` | `/orgs/{orgID}/links?orgId={orgID}` | `links.create` | Create `{"url","slug","title","tags"}` |
| `GET` | `/orgs/{orgID}/links?orgId={orgID}` | `links.list` | List links; filters: `q` (alias prefix), `tags` (any match) |
| `GET` | `/orgs/{orgID}/links/{linkID}?orgId={orgID}` | `links.get` | Get one link |
| `PATCH` | `/orgs/{orgID}/links/{linkID}?orgId={orgID}` | `links.update` | Update `url`, `title`, `tags`, or `disabled` |
| `DELETE` | `/orgs/{orgID}/links/{linkID}?orgId={orgID}` | `links.delete` | Tombstone a link |
| `GET` | `/s/{slug}` | public | Redirect to the destination |

Create body:

```json
{
  "url": "https://example.com/page",
  "slug": "launch",
  "title": "Launch page",
  "tags": ["marketing"]
}
```

`url` must be `http` or `https` and at most 2048 bytes. An omitted `slug`
generates a random 7-character alias. Custom aliases are 3–32 lowercase
letters, digits, or hyphens, start and end alphanumeric, are globally unique,
and cannot be changed after creation. `title` is optional and capped at 120
bytes. Tags use the shared 20-item, 32-byte limits. Duplicate aliases return
`409`.

List responses use `{"items":[],"nextOffset":20}`. `q` matches the start of the
alias; `tags` matches any listed tag. `nextOffset` is `null` on the last page.

`PATCH` accepts any subset of `url`, `title`, `tags`, and `disabled`; at least
one field is required, and `slug` is rejected. `disabled: true` stops the
redirect and `false` re-enables it. `DELETE` tombstones the record: it is
hidden from lists, returns `404` on redirect, and its alias stays reserved.

The redirect is public and returns:

```http
HTTP/1.1 302 Found
Location: https://example.com/page
Cache-Control: no-store
X-Robots-Tag: noindex
```

Missing, disabled, and deleted aliases return `404 {"error":"link not found"}`.
`shortUrl` in management responses is built from the API request origin, so it
uses the Function URL until a custom domain is placed in front of the API.

The `link` feature governs management:

```yaml
link:
  viewer: [links.list, links.get]
  editor: [links.list, links.get, links.create, links.update]
  admin: [links.list, links.get, links.create, links.update, links.delete]
```

### Projects

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/projects?orgId={orgID}` | Create `{"name","currency","tags"}` |
| `GET` | `/projects?orgId={orgID}` | List projects; filters: `tags` (any match) |
| `GET` | `/projects/{projectID}?orgId={orgID}` | Get a project |
| `PATCH` | `/projects/{projectID}?orgId={orgID}` | Partial update of `name` and/or `tags` |

Project `currency` is a three-letter ISO code. Expenses inherit it and cannot
change it. Tags are trimmed, deduplicated, sorted, and capped at 20 items with a
32-byte limit per tag. `tags=food,travel` returns projects matching **any** of
the requested tags. This differs from expense tag filtering, which requires all
tags to match.

There is no project delete endpoint.

### Expenses

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/expenses?orgId={orgID}` | Create an expense |
| `GET` | `/expenses?orgId={orgID}` | List expenses; filters: `projectId`, `projectIds`, `tags`, `tag`, `from`, `to` |
| `GET` | `/expenses/summary?orgId={orgID}` | Totals grouped by project and currency; same filters |
| `GET` | `/expenses/{expenseID}?orgId={orgID}` | Get an expense |
| `PATCH` | `/expenses/{expenseID}?orgId={orgID}` | Update fields |
| `DELETE` | `/expenses/{expenseID}?orgId={orgID}` | Delete an expense |

Expense body fields:

```json
{
  "projectId": "64f...",
  "type": "debit",
  "amount": 1299,
  "description": "Lunch",
  "tags": ["food", "travel"],
  "spentAt": "2026-01-02T12:30:00Z"
}
```

`type` is `debit` for money out or `credit` for money in. If it is omitted on
create, it defaults to `debit` so existing clients keep working. Records without
a type are also treated as debit when read and aggregated. `amount` is a
positive integer in minor units; `type` carries the direction. `currency` is
derived from the project. Tags are trimmed, deduplicated, sorted, and capped at
20 items. `spentAt` is required and uses RFC 3339.

`PATCH /expenses/{expenseID}` accepts any subset of `projectId`, `type`,
`amount`, `description`, `tags`, and `spentAt`; at least one field is required,
and `currency` cannot change.

`GET /expenses` accepts `tags=food,travel` (or repeated `tags` parameters) and
requires every listed tag to match. `tag` remains as a single-tag fallback.
`from` and `to` are inclusive RFC 3339 bounds. `projectId` selects one project;
`projectIds` selects several as a comma-separated list (or repeated parameter),
up to 100 IDs. Do not send both. The list response contains only the current page
and `nextOffset`:

```json
{
  "items": [],
  "nextOffset": 20
}
```

`nextOffset` is `null` on the last page. List responses do not compute totals.

`GET /expenses/summary` accepts the same filters, has no pagination, and returns
totals grouped by project and currency:

```json
{
  "items": [
    {
      "projectId": "64f...",
      "currency": "INR",
      "totalCount": 42,
      "totalDebit": 100000,
      "totalCredit": 123456,
      "netTotal": 23456
    }
  ]
}
```

`totalDebit` and `totalCredit` are sums in minor units. `netTotal` is
`totalCredit - totalDebit`. Amounts are never added across currencies.

### Common behavior

- Error body: `{"error":"message"}`.
- Success statuses: `200`, `201`, `202`, `204`; validation `400`; forbidden
  `403`; missing resource or malformed ObjectID `404`; method not allowed `405`;
  duplicate email/membership and owner-removal conflicts `409`; email delivery
  failure `502`; internal `500`.
- `404` and `405` responses are JSON. `405` responses include the path's
  `Allow` header.
- `HEAD` responses carry headers and no body.
- The Lambda request ID is returned as `X-Request-Id`.
- CORS is handled by `middleware.CORS` using `CORS_ALLOWED_ORIGINS`, which
  defaults to `PUBLIC_BASE_URL`. A `*` entry reflects any non-null origin with
  credentials enabled and is intended for development only.

**Authentication uses a bearer JWT.** Protected routes read the
`Authorization: Bearer` header and trust its `sub` claim after verifying the
signature and expiry; no user lookup is performed for a valid token. The
`pico_auth` cookie is accepted as a fallback. There is no `?userId=` fallback.

**Authorization is declarative.** `internal/authz/resources/routes.yml` maps
each route key to a unit permission (`expenses.list`, `expenses.create`), and
`features.yml` grants those permissions through feature-scoped roles such as
`expense.editor`, `project.viewer`, and `member.admin`. The access token carries
the resolved feature roles; the owner is `["*"]`. `RequireOrgAccess` matches
`?orgId=` and accepts the request when any feature role in the token grants the
route permission. No membership lookup runs per request, so role changes take
effect when the access token is refreshed (at most 30 minutes).

Feature role semantics:

| Feature | viewer | editor | admin |
| --- | --- | --- | --- |
| `org` | read | — | read + update settings |
| `member` | list | — | list + add + update + remove |
| `project` | list + get | + create + update | + create + update |
| `expense` | list + summary + get | + create + update | + delete |
| `link` | list + get | + create + update | + delete |

A user may have different roles per feature in the same organization.

## Data model and indexes

```text
users
  _id, name, email (unique), passwordHash, emailVerified,
  emailVerifiedAt, verificationSentAt, createdAt, updatedAt

orgs
  _id, name, userId (owner), createdAt, updatedAt

memberships
  _id, orgId, userId, roles (feature -> role), createdAt
  unique (orgId, userId)

credentials
  _id, kind (share|api_key), orgId, resourceType, resourceId, projectId,
  scopes, name, tokenHash, createdBy, createdAt,
  expiresAt, revokedAt, lastUsedAt

projects
  _id, orgId, userId, name, currency, tags, createdAt, updatedAt

links
  _id, orgId, userId, slug (unique), url, title, tags,
  disabledAt, deletedAt, createdAt, updatedAt

expenses
  _id, orgId, userId, projectId, amount, currency, description,
  tags, spentAt, createdAt, updatedAt
```

`currency` belongs to the project and is copied into expenses when created.
`amount` is stored in minor units.

`_id` values are MongoDB ObjectIDs. References such as `orgId`, `userId`, and
`projectId` are stored as ObjectID hex strings.

Indexes are created by a command, not during Lambda startup:

```bash
# After configuring .env (see Local development):
make indexes
```

Run `make indexes` after configuring Atlas and before relying on unique email
or membership constraints. The command is idempotent.

## Layout

```text
.
├── AGENTS.md                      # Agent rules and definition of done
├── main.go                        # Lambda entrypoint
├── cmd/
│   ├── indexes/main.go            # Create MongoDB indexes
│   └── server/main.go             # Local development HTTP server
├── docs/
│   ├── architecture.md            # Request flow, boundaries, decisions
│   └── engineering-standards.md   # Clean code, DRY, modularity, testing
├── internal/
│   ├── app/                       # Shared composition root
│   ├── controllers/               # HTTP request/response DTOs and controllers
│   ├── dao/                       # Persistence contracts
│   │   └── mongodb/               # MongoDB stores, indexes, client lifecycle
│   ├── email/                     # SMTP verification email delivery
│   ├── entities/                  # Domain records and sentinel errors
│   ├── httpx/                     # Shared JSON response helpers
│   ├── lambdahttp/                # Function URL <-> net/http adapter
│   ├── middleware/                # Request ID, logging, recovery, identity checks
│   ├── routes/                    # chi router, domain subrouters, 404/405
│   ├── services/                  # Business rules, bcrypt, service inputs
│   └── validation/                # Shared validation errors and pagination
├── test/                          # Tests mirroring the internal packages
├── events/function-url.json       # Sample event for sam local invoke
├── opencode.jsonc                 # Auto-loads docs for OpenCode sessions
├── sample.env                     # Local env template; copy to .env
├── template.yaml                  # Lambda and Function URL resources
├── Makefile
└── samconfig.toml
```

### Authorization policy

| File | Purpose |
| --- | --- |
| `internal/authz/resources/routes.yml` | Route key → method, path, scope. The key path is the unit permission, for example `expenses.list` |
| `internal/authz/resources/features.yml` | Feature → role → unit permissions. Roles are `viewer`, `editor`, and `admin`; the token role is `<feature>.<role>` |

`scope` is `public` (no identity), `user` (identity only), or `org` (identity,
`?orgId=`, a feature role or share scope, and the route permission). The owner is not a declared
role: when an organization's `userId` matches the token subject, issuance writes
`["*"]`, which grants every defined permission. Policy files are embedded in the
binary and validated at startup; `test/routes` fails if a route and the policy
drift.

Access token claim shape:

```json
{
  "sub": "user-123",
  "orgs": {
    "org-personal": ["*"],
    "org-family": ["org.viewer", "project.viewer", "expense.viewer"],
    "org-office": ["org.viewer", "member.viewer", "project.editor", "expense.editor"]
  },
  "type": "access"
}
```

## Requirements

- Go 1.27+
- [SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- Docker (for `sam local invoke`)
- AWS credentials configured for deployment
- A MongoDB Atlas cluster

## MongoDB Atlas setup

1. Create a cluster (the free M0 tier is enough for development).
2. Create a database user with `readWrite` on the `picoservice` database. For
   integration tests, the user also needs access to the test database
   (`readWrite` on `picoservice_test` by default).
3. Add network access for your development machine. Public Lambda egress IPs
   are not static, so a production deployment should run the function in a VPC
   with a NAT gateway and a stable Elastic IP (or use PrivateLink) and allowlist
   that address. For development, `0.0.0.0/0` works but is not a production
   configuration. A VPC Lambda needs NAT to reach the public internet.
4. Copy the `mongodb+srv://...` connection string.
5. Create the indexes with `make indexes`.

The Lambda client is created once per instance and reused across invocations.
It caps the connection pool at 10 connections, so `Atlas connections ≈ Lambda
concurrency × 10`. Account for Atlas connection limits before raising reserved
concurrency.

## Local development

Copy the sample configuration, fill in your Atlas and SMTP values, create the
indexes, then run the same router in a plain HTTP server:

```bash
cp sample.env .env   # default, or use dev.env / prod.env
# Edit the file: MONGODB_URI, JWT_SECRET, SMTP settings, and so on.

make indexes
make run
```

For environment-specific application files:

```bash
cp sample.env dev.env
cp sample.env prod.env
chmod 600 dev.env prod.env

make indexes APP_ENV=dev
make run APP_ENV=dev
make ship APP_ENV=prod
```

`APP_ENV=dev` selects `dev.env` and `dev.deploy.env`; `APP_ENV=prod` selects
`prod.env` and `prod.deploy.env`. With `APP_ENV` unset, `.env` and `deploy.env`
are used. The same selector works for `run`, `indexes`, `test-integration`,
`invoke`, `ship`, and `url`.

Generate a local JWT secret with `openssl rand -hex 32`.

`make run`, `make indexes`, `make test-integration`, and `make invoke` load the
selected application env file automatically. `make deploy` intentionally does
not; pass deployment values explicitly. Exported environment variables still
work for commands that do not load an env file.

Use any SMTP server for development. For example, Mailpit listens on
`localhost:1025` and shows delivered mail at `http://localhost:8025`. With
Amazon SES, use the SES SMTP endpoint on port 587, verify the sender identity,
and verify recipients while the account is in the sandbox.

`.env` is ignored by git. Never commit connection strings, JWT secrets, or SMTP
passwords. `sample.env` contains placeholders only.

A minimal flow:

```bash
# Register a user. The response includes emailVerified:false.
curl -sS -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada","email":"ada@example.com","password":"correct horse"}'

# Open the verification link from Mailpit or your SMTP inbox. It opens the UI.
# The API equivalent is:
curl -sS -X POST http://localhost:8080/auth/verify-email \
  -H 'Content-Type: application/json' \
  -d '{"token":"TOKEN"}'

# Resend the verification email.
curl -sS -X POST http://localhost:8080/auth/verify-email/resend \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com"}'

# Log in and keep the access token.
TOKEN=$(curl -sS -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct horse"}' |
  python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])')

# Create an org owned by that user.
curl -sS -X POST http://localhost:8080/orgs \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Acme"}'

# Create a project in the org.
curl -sS -X POST 'http://localhost:8080/projects?orgId=ORG_ID' \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Launch","currency":"USD","tags":["travel","food"]}'

# List projects matching any of the tags.
curl -sS 'http://localhost:8080/projects?orgId=ORG_ID&tags=travel,food' \
  -H "Authorization: Bearer $TOKEN"

# Record a debit (money out). Currency comes from the project.
curl -sS -X POST 'http://localhost:8080/expenses?orgId=ORG_ID' \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"projectId":"PROJECT_ID","type":"debit","amount":1299,"description":"Lunch","tags":["food"],"spentAt":"2026-01-02T12:30:00Z"}'

# Record a credit (money in).
curl -sS -X POST 'http://localhost:8080/expenses?orgId=ORG_ID' \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"projectId":"PROJECT_ID","type":"credit","amount":5000,"description":"Client payment","spentAt":"2026-01-03T09:00:00Z"}'

# List one project page.
curl -sS 'http://localhost:8080/expenses?orgId=ORG_ID&projectId=PROJECT_ID&tags=food,travel&from=2026-01-01T00%3A00%3A00Z&to=2026-01-31T23%3A59%3A59Z&limit=20&offset=0' \
  -H "Authorization: Bearer $TOKEN"

# Summarize the same filters by project and currency.
curl -sS 'http://localhost:8080/expenses/summary?orgId=ORG_ID&projectId=PROJECT_ID&tags=food,travel&from=2026-01-01T00%3A00%3A00Z&to=2026-01-31T23%3A59%3A59Z' \
  -H "Authorization: Bearer $TOKEN"
```

Set `PORT` to use a different port:

```bash
PORT=3000 make run
```

## Tests and checks

```bash
make check
```

`make check` verifies formatting with `gofmt`, runs `go vet`, and runs all
tests. Use `make test` to run tests only. Tests live under `test/`, mirroring
the `internal/` packages, and use only exported APIs.

The tests cover health routing, JSON 404/405 responses with per-route `Allow`
headers, request-ID propagation, panic recovery, query identity and membership
middleware, the user/org/membership/project/expense services and controllers,
email verification tokens and SMTP messages, and the MongoDB stores.

MongoDB integration tests skip unless a test URI is configured:

```bash
export MONGODB_TEST_URI='mongodb+srv://...'
make test-integration
```

The store integration test clears its collections in `MONGODB_TEST_DATABASE`
(default `picoservice_test`) before and after it runs. The ping integration test
uses the same database setting.

## Engineering standards

Project conventions are documented for both humans and agents:

- [`AGENTS.md`](AGENTS.md) — agent operating rules and definition of done
- [`docs/architecture.md`](docs/architecture.md) — request flow, package
  boundaries, decisions, and known debt
- [`docs/engineering-standards.md`](docs/engineering-standards.md) — clean code,
  DRY, modularity, testing, and HTTP conventions

OpenCode loads the detailed docs automatically through `opencode.jsonc`.

## Build and local Lambda invocation

```bash
make build
make invoke
```

`make invoke` requires Docker and loads the settings from `.env` (or the
current environment). It uses `events/function-url.json` and passes the values
to the local container as template parameters. `SMTP_USERNAME` and
`SMTP_PASSWORD` are optional. The template sets `MONGODB_DATABASE` to
`picoservice`.

## Deploy

```bash
make deploy
```

`make deploy` does not load `.env`. It requires `MONGODB_URI`, `JWT_SECRET`,
`PUBLIC_BASE_URL`, `EMAIL_FROM`, and `SMTP_ADDR`. `SMTP_USERNAME` and
`SMTP_PASSWORD` are optional but must be set together. `CORS_ALLOWED_ORIGINS`
is optional and defaults to `PUBLIC_BASE_URL`. `MongoDBURI`, `JwtSecret`,
`SmtpUsername`, and `SmtpPassword` are `NoEcho` parameters, so CloudFormation
masks them in stack descriptions. To deploy without the Makefile:

```bash
sam deploy --parameter-overrides \
  MongoDBURI='mongodb+srv://...' \
  JwtSecret='...' \
  PublicBaseURL='https://example.com' \
  CorsAllowedOrigins='https://example.com' \
  EmailFrom='noreply@example.com' \
  SmtpAddr='smtp.example.com:587' \
  SmtpUsername='...' \
  SmtpPassword='...'
```

`PUBLIC_BASE_URL` must be the public UI origin, for example
`https://app.example.com`. The API uses it to build verification links and as
the default CORS origin; it is not the Function URL.

### One-command deploy

The repository includes a wrapper for local deployments.

`deploy.env` holds **only AWS deployment settings**:

```bash
cp sample.deploy.env deploy.env   # first time only
chmod 600 deploy.env
# Fill in AWS_PROFILE or AWS keys, AWS_REGION, and STACK_NAME.
# Optionally set S3_BUCKET to a dedicated artifact bucket in the same region.
```

If `S3_BUCKET` is set, the script uses it and deploys with `--s3-bucket` plus
`--no-resolve-s3`, bypassing SAM's managed bucket. If the bucket does not exist,
the script creates it automatically in the stack region. The deploy user needs
`s3:CreateBucket` for the first run, plus `s3:PutObject`, `s3:GetObject`,
`s3:ListBucket`, and `s3:GetBucketLocation` on the bucket. Pre-create the bucket
and use `--no-create-bucket` if the user should not create buckets.

Application settings stay in `.env` (template: `sample.env`):

```bash
chmod 600 .env
# MONGODB_URI, MONGODB_DATABASE, JWT_SECRET, PUBLIC_BASE_URL,
# CORS_ALLOWED_ORIGINS, EMAIL_FROM, SMTP_ADDR, optional SMTP_USERNAME/SMTP_PASSWORD
```

Then deploy:

```bash
make ship
```

Or run `./scripts/deploy.sh` directly. The script:

- requires both env files to exist with mode `600`
- loads the selected application env file and deploy env file
- checks the required values
- verifies AWS credentials with `aws sts get-caller-identity`
- uses a dedicated `S3_BUCKET` when configured, creating it if missing unless
  `--no-create-bucket` is passed
- runs `sam build` and `sam deploy`
- ensures MongoDB indexes unless `--skip-indexes` is passed
- checks `/ping` and `/ready`
- prints the Function URL and log command

Prefer `AWS_PROFILE` with credentials in `~/.aws`; raw
`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` are also supported in `deploy.env`.
Use `--yes` to skip the changeset confirmation, `--watch` to run
`sam sync --watch` after deploying, and `--no-create-bucket` to require a
pre-existing artifact bucket:

```bash
make ship ARGS='--yes'
make ship ARGS='--watch'
make ship ARGS='--no-create-bucket'
```

To keep deployment application values separate from local development, use
`APP_ENV`:

```bash
make ship APP_ENV=prod
```

That loads `prod.env` for application values and `prod.deploy.env` for AWS
settings. `APP_ENV_FILE` and `DEPLOY_ENV_FILE` still work as explicit
overrides.

Real `*.env` files are ignored and contain secrets. Application secrets stay in
the app env file and are never committed.

The one-command script already runs the index and health checks. For the manual
`make deploy` path, run the index command and retrieve the Function URL:

```bash
make indexes
make url
```

Then call the endpoints (the Function URL includes a trailing slash):

```bash
curl -i "$(make url)ping"
curl -i "$(make url)ready"
```

## Configuration

| Variable | Required | Description |
| --- | --- | --- |
| `MONGODB_URI` | yes | MongoDB Atlas connection string |
| `MONGODB_DATABASE` | yes | Database name, `picoservice` in `template.yaml` |
| `JWT_SECRET` | yes | Signs email verification and access tokens; at least 32 bytes |
| `PUBLIC_BASE_URL` | yes | Frontend origin used in verification links and as the default CORS origin |
| `CORS_ALLOWED_ORIGINS` | no | Comma-separated allowed browser origins; defaults to `PUBLIC_BASE_URL`; `*` allows all |
| `EMAIL_FROM` | yes | Sender address for verification email |
| `SMTP_ADDR` | yes | SMTP server address, `host:port` |
| `SMTP_USERNAME` | no | SMTP username; set with `SMTP_PASSWORD` |
| `SMTP_PASSWORD` | no | SMTP password; set with `SMTP_USERNAME` |
| `MONGODB_TEST_URI` | integration tests | Test-cluster connection string |
| `MONGODB_TEST_DATABASE` | no | Integration test database, default `picoservice_test` |
| `PORT` | no | Local server port, default `8080` |

AWS deployment settings live in `deploy.env` or `$(APP_ENV).deploy.env`
(templates: `sample.deploy.env`, `sample.dev.deploy.env`,
`sample.prod.deploy.env`): `AWS_PROFILE` or raw
`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, optional
`AWS_DEFAULT_REGION` (the script derives it from `AWS_REGION` when omitted),
optional `S3_BUCKET` for a dedicated SAM artifact bucket (created automatically
on first deploy), and optional `STACK_NAME` (default `picoservice`). Application
settings never go in a deploy env file.

`make ship` loads the selected application env file (`APP_ENV=prod` uses
`prod.env`) and the selected deploy env file (`prod.deploy.env`). `make deploy`
reads explicit environment values only; local `make` targets use `.env` by
default. See `sample.env`.

- The Function URL is public (`AuthType: NONE`).
- CORS is handled by the application middleware and uses
  `CORS_ALLOWED_ORIGINS`, defaulting to `PUBLIC_BASE_URL`, with credentials
  enabled. `*` allows any non-null origin and is development-only.
- Invocation mode is `BUFFERED`.
- Lambda runs the `provided.al2023` runtime on `x86_64` with a 10 second
  timeout.
- Missing MongoDB configuration fails at startup instead of starting in a
  degraded state. An Atlas outage keeps `/ping` available and makes `/ready`
  return `503`.

To add a route, add any new domain fields to `internal/entities`, add the
request/response structs and controller method in `internal/controllers`, add
the service input and business logic in `internal/services`, add or extend the
DAO contract in `internal/dao` and its MongoDB implementation in
`internal/dao/mongodb`, then register the route in the matching domain
subrouter in `internal/routes` (`routes_users.go`, `routes_orgs.go`, and so
on). `internal/routes/router.go` mounts each subrouter under its prefix. Add
tests under `test/`, update the route contract in `README.md` and
`docs/architecture.md`, and add indexes in
`internal/dao/mongodb/indexes.go` if the route introduces a new query pattern.
