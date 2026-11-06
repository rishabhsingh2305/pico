# Architecture

## Overview

picoservice is a single Go AWS Lambda function exposed through a public Lambda
Function URL. There is no API Gateway. The same chi router is used by the Lambda
handler and by the local development server. MongoDB Atlas stores users,
organizations, memberships, projects, and expenses. Expenses are debit or credit
entries; amounts stay positive and the type carries the direction.

```text
client
  -> Lambda Function URL (payload v2, BUFFERED)
  -> Go Lambda runtime
  -> internal/lambdahttp.Adapter
  -> internal/routes (chi router)
  -> internal/middleware (request ID, logging, recovery, identity checks)
  -> internal/controllers
  -> internal/services
  -> internal/dao interfaces
  -> internal/dao/mongodb stores
  -> JSON response
```

## Packages and responsibilities

| Package | Responsibility | May import |
| --- | --- | --- |
| `main` | Lambda entrypoint; opens MongoDB, calls `app.New` | `internal/app`, `internal/lambdahttp`, `internal/dao/mongodb` |
| `cmd/server` | Local HTTP server and graceful shutdown | `internal/app`, `internal/dao/mongodb` |
| `cmd/indexes` | Create MongoDB indexes | `internal/dao/mongodb` |
| `internal/app` | Composition root: DAO implementations, services, routes | `internal/dao/mongodb`, `internal/email`, `internal/services`, `internal/routes` |
| `internal/routes` | chi setup, middleware wiring, domain subrouters, per-route `Allow` | `internal/controllers`, `internal/middleware`, `internal/httpx`, chi |
| `internal/controllers` | HTTP parsing, status codes, request/response DTOs, response mapping | `net/http`, `context`, `internal/services`, `internal/middleware`, `internal/entities`, `internal/httpx`, `internal/validation` |
| `internal/services` | Business rules, bcrypt, email verification, service input structs, narrow lookup interfaces | stdlib, `internal/dao`, `internal/entities`, `internal/validation`, `golang.org/x/crypto/bcrypt`, `github.com/golang-jwt/jwt/v5` |
| `internal/dao` | Per-domain persistence contracts and query/update structs | `internal/entities`, stdlib |
| `internal/dao/mongodb` | MongoDB client lifecycle, stores, index creation, readiness ping | MongoDB driver, `internal/dao`, `internal/entities`, stdlib |
| `internal/email` | SMTP delivery and multipart verification messages | stdlib |
| `internal/entities` | Domain records and sentinel errors | stdlib |
| `internal/validation` | Shared validation error, name checks, pagination | stdlib |
| `internal/httpx` | JSON response helpers, request-ID context, error envelope | stdlib |
| `internal/lambdahttp` | Function URL event ↔ `net/http`; cookies, base64 bodies, response conversion | `events`, `net/http`, `internal/httpx` |
| `internal/middleware` | CORS, request ID propagation, structured request logging, panic recovery, query identity and membership checks | stdlib, chi middleware, `internal/httpx` |
| `test/*` | External tests mirroring the `internal` packages | `internal/*`, stdlib, `httptest`; MongoDB driver for integration fixtures |

Tests live under `test/` and use only exported APIs. Integration tests are gated
by environment variables and skip by default.

Dependency rules:

- `lambdahttp` must not import `routes`, `controllers`, `services`, `dao`, or
  `entities`.
- `routes` must not import `dao` or the MongoDB driver; it wires middleware,
  controllers, and service lookups through `routes.Deps`.
- `middleware` must not import `routes`, `controllers`, `services`, `dao`, or
  `entities`; it defines small lookup interfaces and depends on `internal/httpx`
  and chi middleware.
- `controllers` must not import `routes`, `lambdahttp`, `dao`, `dao/mongodb`, or
  the MongoDB driver. They depend on small consumer-side service interfaces.
- `services` must not import HTTP, AWS, or MongoDB driver packages. They consume
  `dao` contracts and narrow lookup interfaces.
- `dao` must not import `services`, `controllers`, `routes`, or the MongoDB
  driver. It contains contracts and query/update structs only.
- `dao/mongodb` must not import `controllers`, `routes`, or `services`.
- `email` implements the mailer interface consumed by `services`; it must not
  import `controllers`, `routes`, `services`, `dao`, or `entities`.
- `httpx` and `validation` must not import feature packages.
- `entities` imports stdlib only.
- AWS-specific types must not leak past `lambdahttp`. MongoDB driver types stay
  in `dao/mongodb`; integration tests may use the driver only to isolate and
  drop test databases.
- `internal/app` is the shared composition root. `main` and `cmd` wire only the
  MongoDB client and the application handler.

## Data model

```text
users
  _id                 ObjectID
  name                string
  email               string (lowercase, unique)
  passwordHash        string (bcrypt)
  emailVerified       bool
  emailVerifiedAt     time | null
  verificationSentAt  time | null
  createdAt           time
  updatedAt           time

orgs
  _id          ObjectID
  name         string
  userId       string (owner)
  createdAt    time
  updatedAt    time

memberships
  _id          ObjectID
  orgId        string
  userId       string
  createdAt    time
  unique (orgId, userId)

projects
  _id          ObjectID
  orgId        string
  userId       string
  name         string
  currency     string (3-letter ISO code)
  tags         []string
  createdAt    time
  updatedAt    time

expenses
  _id          ObjectID
  orgId        string
  userId       string
  projectId    string
  type         string (debit|credit; missing means debit)
  amount       int64 (positive minor units)
  currency     string (3-letter ISO code)
  description  string
  tags         []string
  spentAt      time
  createdAt    time
  updatedAt    time
```

`_id` values are MongoDB ObjectIDs. References (`orgId`, `userId`, `projectId`)
are stored as ObjectID hex strings. This keeps filters and indexes consistent
with API identifiers.

Organization ownership is `orgs.userId`. Membership roles are derived, not
stored: the owner is `owner`; every other membership is `member`.

## API contract

All responses are JSON unless the status is `204`. Error responses use
`{"error":"message"}`. Read endpoints support `GET` and `HEAD`. List endpoints
accept `limit` and `offset` (default 20, maximum 100).

### Health

| Method | Path | Status | Body |
| --- | --- | --- | --- |
| GET | `/ping` | 200 | `{"message":"pong"}` |
| HEAD | `/ping` | 200 | empty |
| GET | `/ready` | 200 | `{"status":"ok"}` |
| GET | `/ready` | 503 | `{"error":"database unavailable"}` |
| HEAD | `/ready` | 200/503 | empty |

### Authentication

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/auth/login` | 200 | `{"email","password"}`; returns `{"accessToken","tokenType","user","expiresAt"}` and also sets the `pico_auth` cookie fallback |
| POST | `/auth/logout` | 204 | Clears the auth cookie; the client clears its stored token |
| GET | `/auth/session` | 200 | Returns `{"user","expiresAt"}` from `Authorization: Bearer` or the cookie; `401` when neither is valid |

Access tokens are HS256 JWTs valid for 7 days. Claims are `sub` (user ID),
`iss=picoservice`, `aud=picoservice-api`, `type=access`, `iat`, and `exp`.
Login requires a verified email. Wrong credentials return `401 invalid email or
password`; unverified accounts return `403 email not verified`. Tokens are
stateless: a valid token requires no user lookup, and logout does not revoke an
already-issued token.

### Users

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/users` | 201 | `{"name","email","password"}`; creates an unverified user and sends a verification email; `Location: /users/{id}` |
| GET | `/users?userId=actor` | 200 | Paginated list |
| GET | `/users/{userID}?userId=actor` | 200 | |
| PATCH | `/users/{userID}?userId=actor` | 200 | Self only; `name` and/or `email`; changing email resets verification |
| POST | `/users/{userID}/password?userId=actor` | 204 | Self only; `currentPassword`, `newPassword` |

No user delete. Password hashes are never returned. Responses include
`emailVerified`. Duplicate email returns `409`. Protected user routes use
`Authorization: Bearer` (or the cookie fallback); the `userId` query parameter
is a temporary fallback.

### Email verification

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/auth/verify-email` | 200 | `{"token":"..."}`; validates the JWT, marks the email verified, and returns `status`, `userId`, and `email` |
| POST | `/auth/verify-email/resend` | 202 | `{"email"}`; generic for unknown or verified addresses and throttled to one send per 60 seconds |

Verification tokens are HS256 JWTs valid for 24 hours and are sent as links in
email. Verification is idempotent. Invalid, expired, or mismatched tokens
return `400` with a generic message. Resend requests for unknown or already
verified addresses still return `202` to avoid enumeration.

### Organizations

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/orgs?userId=actor` | 201 | `{"name"}`; creates owner membership |
| GET | `/orgs?userId=actor` | 200 | Orgs where actor is a member |
| GET | `/orgs/{orgID}?orgId=actor&userId=actor` | 200 | Member only |
| PATCH | `/orgs/{orgID}?orgId=actor&userId=actor` | 200 | Owner only; `{"name"}` |

Path-scoped org routes repeat `orgId` in the query because `RequireMember`
reads the tenant from the query. The actor comes from the bearer token or
cookie fallback.

No org delete.

### Memberships

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/orgs/{orgID}/members?orgId=actor&userId=actor` | 201 | Owner only; `{"userId"}` |
| GET | `/orgs/{orgID}/members?orgId=actor&userId=actor` | 200 | Member only; paginated |
| DELETE | `/orgs/{orgID}/members/{memberUserID}?orgId=actor&userId=actor` | 204 | Owner only; owner cannot be removed |

Membership responses include a derived `role` of `owner` or `member`.

### Projects

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/projects?orgId=&userId=` | 201 | `{"name","currency","tags"}` |
| GET | `/projects?orgId=&userId=` | 200 | Paginated list; `tags` filter uses any match |
| GET | `/projects/{projectID}?orgId=&userId=` | 200 | |
| PATCH | `/projects/{projectID}?orgId=&userId=` | 200 | Partial `{"name"}` and/or `{"tags"}` |

Project `currency` is immutable after creation. Expenses inherit it. Project
tags are trimmed, deduplicated, sorted, and capped at 20 items with a 32-byte
limit per tag. `tags=food,travel` returns projects matching either tag; expense
tag filtering remains all-match.

No project delete.

### Expenses

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/expenses?orgId=&userId=` | 201 | Create |
| GET | `/expenses?orgId=&userId=` | 200 | Filters: `projectId`, `tags`, `tag`, `from`, `to`; returns page and totals |
| GET | `/expenses/{expenseID}?orgId=&userId=` | 200 | |
| PATCH | `/expenses/{expenseID}?orgId=&userId=` | 200 | Partial update |
| DELETE | `/expenses/{expenseID}?orgId=&userId=` | 204 | |

Expense `type` is `debit` for money out or `credit` for money in. It defaults to
debit when omitted on create, and records without a type are read and aggregated
as debit. Expense `amount` is a positive integer in minor units; the type carries
direction. `currency` is inherited from the project and cannot be changed through
the expense API. Tags are trimmed, deduplicated, sorted, and capped at 20 items.
`spentAt` is required.

`tags=food,travel` (or repeated `tags` parameters) requires every tag to match.
`tag` remains as a single-tag fallback. `from` and `to` are inclusive RFC 3339
bounds. The response includes totals over every expense matching the filters,
independent of pagination:

```json
{
  "items": [],
  "totalCount": 42,
  "totalDebit": 100000,
  "totalCredit": 123456,
  "netTotal": 23456
}
```

`totalDebit` and `totalCredit` are in minor units. `netTotal` is
`totalCredit - totalDebit`. The totals are meaningful as a single amount when
`projectId` narrows the result to one project currency.

### Errors

| Status | Meaning |
| --- | --- |
| 400 | Validation error, malformed JSON, missing `orgId` query parameter, invalid verification token |
| 401 | Missing, invalid, or expired auth token; invalid login credentials |
| 403 | Email not verified, actor is not a member, or a non-owner attempted an owner-only action |
| 404 | Resource missing or malformed ObjectID |
| 405 | Method not allowed; includes `Allow` |
| 409 | Duplicate email/membership, owner removal, or conflict |
| 502 | Verification email delivery failed |
| 500 | Unexpected error; details are logged, not returned |

Invalid ObjectIDs are treated as not found rather than as a separate error.

## Identity and tenancy

Protected routes authenticate with `Authorization: Bearer <token>`, falling back
to the `pico_auth` cookie:

- `RequireAuth` validates the access JWT and stores its `sub` as the actor. A
  valid token requires no database lookup.
- Requests without either credential fall back to the temporary `userId` query
  parameter and verify that the user exists. A present but invalid bearer token
  or cookie never falls back.
- `RequireMember` reads the actor from the request context and the tenant from
  the `orgId` query parameter, then verifies the organization and membership
  before calling the controller. Missing `orgId` returns `400`; missing orgs
  return `404`; non-members return `403`.

The query fallback is temporary and spoofable. Remove it once the UI always
sends a token.

Owner-only operations are enforced in the org and membership services by
comparing the actor to `orgs.userId`. Creating an org inserts the org and then
the owner membership. If the membership insert fails, the org is deleted as a
best-effort cleanup.

## Email verification

- `POST /users` creates the user with `emailVerified: false`, signs a JWT, and
  sends it over SMTP. The link points at
  `{PUBLIC_BASE_URL}/verify-email#token=...`, where `PUBLIC_BASE_URL` is the UI
  origin. The UI reads the fragment and calls `POST /auth/verify-email` with the
  token in the JSON body.
- Tokens use HS256 with `JWT_SECRET` (at least 32 bytes). Claims are `sub`
  (user ID), `email`, `purpose=email_verification`, issuer, audience, `iat`, and
  `exp`. Tokens expire after 24 hours.
- Verification loads the user, compares the token email with the current email,
  and marks the user verified. Replaying a valid link is idempotent.
- Changing the email address clears verification and sends a new link. If
  delivery fails, the new address remains stored and unverified; resend can
  retry.
- Signup deletes the new user as a best-effort cleanup when the email cannot be
  sent, so the caller can retry.
- Resend is throttled to one send per 60 seconds and always returns `202` for
  unknown or verified addresses to prevent enumeration.
- SMTP uses STARTTLS when the server advertises it. `SMTP_USERNAME` and
  `SMTP_PASSWORD` are optional and must be set together.

## Passwords

- Passwords are 8–72 bytes (the bcrypt limit).
- Only bcrypt hashes are stored. Plaintext passwords are never stored, logged,
  or returned.
- `POST /users` hashes the password. `POST /users/{id}/password` verifies the
  current password before storing a new hash.
- Login verifies the bcrypt hash and issues a stateless access token. Unknown
  emails run a dummy bcrypt comparison so timing does not reveal whether an
  account exists.
- Email verification JWTs prove email ownership only and use a different
  audience from access tokens, so they cannot authorize API calls.

## MongoDB

- `MONGODB_URI` and `MONGODB_DATABASE` are required. Missing configuration fails
  at startup; this is a deployment error, not an Atlas outage.
- `JWT_SECRET`, `PUBLIC_BASE_URL`, `EMAIL_FROM`, and `SMTP_ADDR` are also
  required at startup. `PUBLIC_BASE_URL` is the UI origin used in verification
  links and as the default CORS origin. `CORS_ALLOWED_ORIGINS` is optional and
  overrides that origin list. `SMTP_USERNAME` and `SMTP_PASSWORD` are optional
  and must be set together.
- The client is created once during process startup and reused. It is never
  created per request. `mongo.Connect` does not validate reachability, so
  startup does not depend on Atlas being available. An SRV connection string
  still requires a DNS lookup during client creation, so a DNS failure prevents
  startup.
- `/ready` applies a 3 second timeout, pings the primary, and returns `503` on
  failure. The driver uses a 5 second server-selection timeout and a 5 second
  connect timeout.
- The connection pool is capped at 10 connections per client. Total Atlas
  connections are roughly Lambda concurrency × 10.
- `/ping` does not touch MongoDB, so liveness stays available during an Atlas
  outage.
- Driver errors are logged with the request ID; clients only see stable
  messages.
- Services use 3 second operation timeouts. Identity middleware uses one
  shared 3 second budget for its user/org/membership lookups.

### Indexes

Indexes are created by `(*mongodb.Client).EnsureIndexes` in
`internal/dao/mongodb` through `cmd/indexes` and `make indexes`, not during
Lambda startup. They are required for correctness,
not only performance:

```text
users:        {email:1} unique
              {createdAt:-1, _id:-1}
orgs:         {userId:1, createdAt:-1, _id:-1}
memberships:  {orgId:1, userId:1} unique
              {userId:1, createdAt:-1, _id:-1}
              {orgId:1, createdAt:-1, _id:-1}
projects:     {orgId:1, createdAt:-1, _id:-1}
              {orgId:1, tags:1}
expenses:     {orgId:1, spentAt:-1, _id:-1}
              {orgId:1, projectId:1, spentAt:-1, _id:-1}
              {orgId:1, tags:1}
```

The unique indexes on `users.email` and `(memberships.orgId,
memberships.userId)` enforce duplicate prevention. The commands are
idempotent.

### Transactions

Multi-document writes are not transactional. Org creation cleanup is
best-effort. Membership changes and expense updates are single-document writes.
Revisit transactions if cross-collection invariants become stricter.

### Atlas networking

A public Lambda reaches Atlas over the internet, but its egress IPs are not
static. For development, `0.0.0.0/0` is the practical option. For production,
run the function in a VPC with a NAT gateway and a stable Elastic IP (or use
PrivateLink) and allowlist that address. A VPC Lambda needs NAT to reach the
public internet.

### Secrets

The connection string is supplied as the `MongoDBURI` CloudFormation parameter
with `NoEcho: true` and injected as `MONGODB_URI`. `JwtSecret` and the SMTP
credentials use `NoEcho` parameters as well. Secrets Manager and the AWS
Parameters and Secrets Lambda Extension are the intended next step; do not add
a caching library that is pinned to AWS SDK v1.

## Lambda Function URL

- Payload format v2, `InvokeMode: BUFFERED`.
- Public access: `AuthType: NONE`.
- CORS is handled by `middleware.CORS` using `CORS_ALLOWED_ORIGINS`, which
  defaults to `PUBLIC_BASE_URL`. A `*` entry reflects any non-null origin and is
  intended for development only. The Function URL has no CORS configuration;
  adding both would duplicate `Access-Control-Allow-*` headers.
- Buffered responses are limited to 6 MB.
- Cookies are carried in the event/response `cookies` array, not the normal
  headers map.
- Request bodies are decoded from base64 by the adapter when needed. Malformed
  events and bodies are logged with the request ID and returned as a JSON 400.
- Response bodies are sent as text when the content type is textual and the body
  is valid UTF-8. Non-textual content types (for example
  `application/octet-stream`) and invalid UTF-8 bodies are base64 encoded with
  `isBase64Encoded: true`.
- HEAD responses are sent without a body, matching `net/http` behavior.

## Runtime and deployment

- Runtime: `provided.al2023`, `x86_64`
- Timeout: 10 seconds; memory: 128 MB
- Build method: SAM `go1.x` (runs `go build` at the repository root)
- Deployment: `sam deploy` using `samconfig.toml` (stack `picoservice`).
  `scripts/deploy.sh` / `make ship` wraps `sam build`, `sam deploy`, index
  creation, and health checks. It loads the selected application env file
  (`APP_ENV=prod` uses `prod.env`, unset uses `.env`) and the selected deploy
  env file (`prod.deploy.env` or `deploy.env`; ignored, mode `600`, templates
  `sample.prod.deploy.env` / `sample.deploy.env`). `make deploy` remains the
  explicit-env fallback and does not load env files.
- Environment: `MONGODB_URI` (parameter), `MONGODB_DATABASE`, `JWT_SECRET`,
  `PUBLIC_BASE_URL`, optional `CORS_ALLOWED_ORIGINS`, `EMAIL_FROM`, `SMTP_ADDR`,
  and optional `SMTP_USERNAME` / `SMTP_PASSWORD`
- Local development: `cmd/server` on `PORT` (default `8080`). `make run`,
  `make indexes`, `make test-integration`, and `make invoke` load the selected
  application env file (`APP_ENV=dev` uses `dev.env`); `sample.env` is the
  committed template.
- Index creation: `make indexes` (loads the selected app env) or the deploy
  script after it loads the application env file; requires `MONGODB_URI` and
  `MONGODB_DATABASE`
- Local Lambda invocation: `make invoke` (loads the selected app env; requires
  Docker)
- Deployment convenience: `deploy.env` or `$(APP_ENV).deploy.env` contains only
  AWS/deployment settings; application secrets stay in the selected app env
  file. The script refuses world-readable files and validates required values
  before deploying.

## Decisions

### Function URL instead of API Gateway

The service has no gateway-specific needs such as stages, WAF, request
validation, or API keys. A Function URL is simpler and cheaper. Revisit this
decision if authentication, throttling, or other gateway features are needed.

### Custom buffered adapter instead of a framework adapter

`aws-lambda-go-api-proxy` was archived in May 2025 and does not support
`events.LambdaFunctionURLRequest`. The official `lambdaurl` package supports
streaming-only Function URLs, which complicates local invocation and CORS. The
small adapter in `internal/lambdahttp` keeps buffered mode, supports
`sam local invoke`, and is covered by tests.

### chi instead of a full framework or stdlib-only routing

chi provides lightweight routing and middleware while keeping controllers as
plain `net/http` handlers. Revisit this decision if the route set shrinks and
the dependency no longer pays for itself.

Domain route files build a `chi.Router` with relative paths, and
`routes/router.go` mounts each one under its prefix (`/users`, `/orgs`, and so
on). This keeps prefixes and composition in one place while letting each domain
file stay small. Because chi's mount stubs match every method, the JSON 405
handler derives `Allow` from a flat probe router built by walking the
application route tree.

### No RealIP middleware

The adapter sets `RemoteAddr` from the Function URL event's trusted source IP.
The local server uses the connection address. `middleware.RealIP` was removed
because it trusts client-controlled `X-Forwarded-For` values.

### Official MongoDB driver, no ODM

`go.mongodb.org/mongo-driver/v2` is the maintained official driver and provides
pooling, retries, TLS, and `mongodb+srv` support. ODM and generic repository
libraries hide the options that matter for Lambda (pool limits, timeouts) and
add a second abstraction over a small surface. DAO contracts live in
`internal/dao`; concrete stores live in `internal/dao/mongodb`.

### DAO contracts with narrow service lookups

`internal/dao` defines one persistence contract per domain, and
`internal/dao/mongodb` implements them. Services consume those contracts for
their own persistence and define narrow lookup interfaces for cross-domain
checks (for example, expenses only need `ProjectLookup.FindByID` to read the
project currency and verify the project). Controllers
define small service interfaces. `internal/app` wires the concrete
implementations. This keeps HTTP, MongoDB, and business rules independently
testable.

### Stateless JWT access tokens with temporary query fallback

Login issues an HS256 access JWT returned in the response body and also sets an
HttpOnly cookie fallback. The browser client stores the token in `localStorage`
and sends it as `Authorization: Bearer`. Token validation is local, so protected
requests with a valid token do not query the user collection. Tokens expire
after 7 days and cannot be revoked before expiry; logout clears the client token
and the cookie. Refresh tokens, rotation, and server-side revocation are
intentionally deferred.

The `userId` query parameter remains as a temporary fallback for clients that
have not migrated. It is spoofable and must be removed once the UI always sends
a token.

### Separate index command

Creating indexes during Lambda cold start would add latency and require broad
startup privileges. `make indexes` makes index changes explicit and lets
deployment verify uniqueness separately.

### Environment variables before Secrets Manager

The connection string is a deployment parameter for now. Secrets Manager adds
IAM, SDK, and cold-start cost that is not justified before the first real
deployment. The Lambda Parameters and Secrets Extension is preferred over an
in-process SDK cache when that changes.

Local development uses `.env` (ignored by git) with `sample.env` as the
committed template. `APP_ENV=dev` selects `dev.env` and `dev.deploy.env`;
`APP_ENV=prod` selects `prod.env` and `prod.deploy.env`; unset uses `.env` and
`deploy.env`. The Makefile sources the selected app env for local commands; the
Go binaries still read only environment variables. `make deploy` never loads an
env file, so local values cannot be deployed accidentally.

Deployment convenience uses two files. Application values stay in the selected
app env file (ignored), with `sample.env` committed. AWS deployment settings
stay in `deploy.env` or `$(APP_ENV).deploy.env` (ignored, mode `600`), with
`sample.deploy.env` and per-environment samples committed.
`make ship`/`scripts/deploy.sh` is the only target that loads both; application
secrets are never placed in a deploy env file. The recommended form uses
`AWS_PROFILE`, keeping actual AWS credentials in `~/.aws`.

### Stateless JWT email verification

Verification links use short-lived HS256 JWTs instead of storing token rows.
The token carries the user ID and email, and the database remains the source of
truth: verification compares the claim with the current user record and only
marks the email verified. This avoids a new collection and cleanup job.
Revocation before expiry is not supported; changing the email invalidates old
links through the email comparison.

### SMTP via the standard library

`net/smtp` avoids an AWS SDK dependency and works with SES SMTP, Mailpit, and
other providers. STARTTLS is used when advertised, and credentials are optional
for local development. The tradeoff is that SMTP credentials must be supplied
as deployment parameters, and there is no provider-specific delivery API.

### Project-owned currency

Currency is stored on the project and copied into each expense at creation.
This keeps all expenses in one project consistent and removes the currency choice
from expense creation. Project currency is immutable after creation; changing it
would require a migration policy for existing expenses.

### Project tags use any-match filtering

Project tags are labels for organizing work, so selecting multiple tags broadens
the result set: a project matches when any selected tag is present. Expense tags
describe a transaction and use all-match filtering instead. The two query
contracts are intentionally different and must stay documented.

### Server-side expense totals

Expense totals are computed by MongoDB with the same match filters as the list
but without `limit` or `offset`. Clients never sum the current page, so the
displayed total does not change when the user paginates. The summary uses a
`$group` aggregation for count, debit amount, and credit amount; `netTotal` is
computed as credit minus debit. An empty match returns zero totals. A legacy
document with no `type` is counted as debit through `$ifNull`.

## Known debt

- **Access tokens cannot be revoked.** Logout clears the client token and
  cookie, but a copied token stays valid for up to 7 days, and password changes
  do not invalidate existing tokens.
- **Query identity fallback remains spoofable.** Requests without a token or
  cookie may still supply `userId`. Remove the fallback once the UI migrates.
- Login has no rate limiting or lockout, so online password guessing is
  unthrottled.
- Verification links carry the JWT in the query string. Application request
  logs omit query strings, but upstream access logs may retain them.
- Resend is throttled per user, but signup and resend have no global rate
  limiting or abuse protection.
- Existing users created before these fields are treated as unverified; there
  is no migration or backfill command.
- `JWT_SECRET` and SMTP credentials are CloudFormation parameters, not Secrets
  Manager values.
- `GET /users` is a global directory. It should become admin-only or scoped to
  shared organizations once authentication and roles exist.
- `GET /expenses` returns debit, credit, and net totals. Organization-wide
  totals across projects with different currencies are not meaningful; they
  would need totals grouped by currency. The UI currently requests
  project-scoped totals.
- No user or org delete HTTP endpoints, so deletion/cascade rules are
  unresolved. The user DAO delete exists only for signup cleanup.
- Multi-collection operations are not transactional; org creation cleanup is
  best-effort.
- Unique indexes must be created with `make indexes`; they are not automatic.
- There is no CI pipeline; `make check` is run manually before deployment.
- Atlas connections scale with Lambda concurrency × pool size. There is no
  reserved concurrency or global connection budget yet; revisit before traffic
  grows.
- There are no migrations or schema management yet. Existing projects without a
  `currency` field are read as `USD`; backfill before relying on mixed currencies.
  Existing expenses without a `type` field are read and aggregated as debit.
- The function runs outside a VPC, so Atlas access currently relies on a broad
  IP allowlist; production should use a VPC with NAT/PrivateLink.
- The Function URL is public and unauthenticated.
- The connection string is a CloudFormation parameter rather than a Secrets
  Manager value.

## Extending the service

- **New route:** add or extend the entity in `internal/entities`, add
  request/response DTOs and a controller method in `internal/controllers`, add
  the service input and business logic in `internal/services`, add the DAO
  contract in `internal/dao` and its MongoDB implementation in
  `internal/dao/mongodb`, then register the route in the matching domain
  subrouter in `internal/routes` (`routes_users.go`, `routes_auth.go`,
  `routes_orgs.go`, and so on). `routes/router.go` mounts each subrouter under
  its prefix. Wrap org-scoped routes with `middleware.RequireMember` and
  user-scoped routes with `middleware.RequireAuth`.
- **Business logic:** add a service in `internal/services` and keep controllers
  thin. Controllers call services; services do not know about HTTP.
- **Persistence or AWS calls:** define the contract in `internal/dao` and
  implement it in `internal/dao/mongodb` or another adapter. Do not call the
  MongoDB driver or AWS SDKs from controllers.
- **Email:** implement the `services.Mailer` interface and wire it in
  `internal/app`. Keep message construction in `internal/email`; do not put
  SMTP calls in services or controllers.
- **New index:** add it to `internal/dao/mongodb/indexes.go` and run
  `make indexes`.
- **New dependency:** justify it in the change description and prefer
  stdlib/chi/the official driver. `github.com/golang-jwt/jwt/v5` is approved for
  email verification tokens.
