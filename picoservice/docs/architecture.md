# Architecture

## Overview

picoservice is a single Go AWS Lambda function exposed through a public Lambda
Function URL. There is no API Gateway. The same chi router is used by the Lambda
handler and by the local development server. MongoDB Atlas stores users,
organizations, memberships, projects, expenses, and short links. Expenses are
debit or credit entries; amounts stay positive and the type carries the
direction. Short links redirect publicly; their management is organization
scoped.

```text
client
  -> Lambda Function URL (payload v2, BUFFERED)
  -> Go Lambda runtime
  -> internal/lambdahttp.Adapter
  -> internal/routes (chi router)
  -> internal/middleware (request ID, logging, recovery, auth and org access)
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
| `internal/app` | Composition root: DAO implementations, services, policy, routes | `internal/authz`, `internal/dao/mongodb`, `internal/email`, `internal/services`, `internal/routes` |
| `internal/authz` | Embedded route and feature-role policy: YAML loading, validation, permission lookup | `internal/entities`, stdlib, `gopkg.in/yaml.v3` |
| `internal/routes` | chi setup, middleware wiring, domain subrouters, per-route `Allow` | `internal/controllers`, `internal/middleware`, `internal/authz`, `internal/httpx`, chi |
| `internal/controllers` | HTTP parsing, status codes, request/response DTOs, response mapping | `net/http`, `context`, `internal/services`, `internal/middleware`, `internal/entities`, `internal/httpx`, `internal/validation` |
| `internal/services` | Business rules, bcrypt, email verification, service input structs, narrow lookup interfaces | stdlib, `internal/dao`, `internal/entities`, `internal/validation`, `golang.org/x/crypto/bcrypt`, `github.com/golang-jwt/jwt/v5` |
| `internal/dao` | Per-domain persistence contracts and query/update structs | `internal/entities`, stdlib |
| `internal/dao/mongodb` | MongoDB client lifecycle, stores, index creation, readiness ping | MongoDB driver, `internal/dao`, `internal/entities`, stdlib |
| `internal/email` | SMTP delivery and multipart verification messages | stdlib |
| `internal/entities` | Domain records and sentinel errors | stdlib |
| `internal/validation` | Shared validation error, name checks, pagination | stdlib |
| `internal/httpx` | JSON response helpers, request-ID context, error envelope | stdlib |
| `internal/lambdahttp` | Function URL event ↔ `net/http`; cookies, base64 bodies, response conversion | `events`, `net/http`, `internal/httpx` |
| `internal/middleware` | CORS, request ID propagation, structured request logging, panic recovery, JWT authentication, declarative org authorization | stdlib, chi, `internal/authz`, `internal/httpx` |
| `test/*` | External tests mirroring the `internal` packages | `internal/*`, stdlib, `httptest`; MongoDB driver for integration fixtures |

Tests live under `test/` and use only exported APIs. Integration tests are gated
by environment variables and skip by default.

Dependency rules:

- `lambdahttp` must not import `routes`, `controllers`, `services`, `dao`, or
  `entities`.
- `routes` must not import `dao` or the MongoDB driver; it wires middleware,
  controllers, and service lookups through `routes.Deps`.
- `middleware` must not import `routes`, `controllers`, `services`, `dao`, or
  `entities`; it defines a small access-token interface and depends on
  `internal/authz`, `internal/httpx`, and chi.
- `authz` must not import `routes`, `middleware`, `controllers`, `services`, or
  `dao`. It is pure policy: YAML types, validation, and role/permission checks;
  it depends on `entities` only for the feature and role vocabulary.
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
  verificationOTPHash string
  verificationOTPExpiresAt time | null
  verificationOTPAttempts int
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
  roles        object (feature -> viewer|editor|admin; ignored for the owner)
  createdAt    time
  unique (orgId, userId)

credentials
  _id          ObjectID
  kind         share | api_key
  orgId        string
  resourceType org | project | expense | null
  resourceId   string | null
  projectId    string | null
  scopes       []string
  name         string
  tokenHash    string
  createdBy    string
  createdAt    time
  expiresAt    time | null
  revokedAt    time | null
  lastUsedAt   time | null
  unique (tokenHash)

projects
  _id          ObjectID
  orgId        string
  userId       string
  name         string
  currency     string (3-letter ISO code)
  tags         []string
  createdAt    time
  updatedAt    time

links
  _id          ObjectID
  orgId        string
  userId       string
  slug         string (lowercase, globally unique)
  url          string
  title        string
  tags         []string
  disabledAt   time | null
  deletedAt    time | null
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

Organization ownership is `orgs.userId`. The owner is implicit: it is not stored
as a membership role and token issuance writes `["*"]` for owned organizations.
Every other membership stores a feature-to-role map such as
`{"org":"viewer","project":"editor","expense":"viewer","link":"viewer"}`.
Memberships created
before feature roles existed decode the legacy `role` field: `org.editor` maps to
the default member roles and `org.viewer` maps to read-only roles.

## API contract

All responses are JSON unless the status is `204` or the route is the public
short-link redirect (`302`). Error responses use `{"error":"message"}`. Read
endpoints support `GET` and `HEAD`. List endpoints accept `limit` and `offset`
(default 20, maximum 100).

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
| POST | `/auth/login` | 200 | `{"email","password"}`; returns `accessToken`, `refreshToken`, `tokenType`, `user`, `expiresAt`, and `refreshExpiresAt`; sets `pico_auth` and `pico_refresh` cookie fallbacks |
| POST | `/auth/refresh` | 200 | `{"refreshToken":"..."}` or the `pico_refresh` cookie; returns a new token pair |
| POST | `/auth/logout` | 204 | Clears both auth cookies; the client clears its stored tokens |
| GET | `/auth/session` | 200 | Returns `{"accessToken","user","expiresAt"}` from `Authorization: Bearer` or the cookie; `401` when neither is valid |

Access tokens are HS256 JWTs valid for 30 minutes. Claims are `sub` (user ID),
`iss=picoservice`, `aud=picoservice-api`, `type=access`, `orgs`, `jti`, `iat`,
and `exp`. Refresh tokens use `type=refresh` and are valid for 30 days.
Refreshing re-queries memberships, so a role change takes effect within one
access-token lifetime. Login requires a verified email. Wrong credentials return
`401 invalid email or password`; unverified accounts return `403 email not
verified`. Tokens are stateless: a valid token requires no user lookup, logout
does not revoke an already-issued token, and refresh rotation does not invalidate
older refresh tokens.

### Users

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/users` | 201 | `{"name","email","password"}`; creates an unverified user and sends a verification email |
| GET | `/users/me` | 200 | The authenticated user |
| PATCH | `/users/me` | 200 | `name` and/or `email`; changing email resets verification |
| POST | `/users/me/password` | 204 | `currentPassword`, `newPassword` |

No user delete and no global user directory. Password hashes are never returned.
Responses include `emailVerified`. Duplicate email returns `409`. The actor is
the token `sub`; there is no `?userId=` identity fallback.

### Email verification

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/auth/verify-email` | 200 | `{"token":"..."}`; validates the JWT, marks the email verified, and returns `status`, `userId`, and `email` |
| POST | `/auth/verify-email/otp` | 200 | `{"email":"...","code":"123456"}`; validates the code and returns `status`, `userId`, and `email` |
| POST | `/auth/verify-email/resend` | 202 | `{"email"}`; returns `{"status":"sent","retryAfterSeconds":60}` and `Retry-After: 60`; generic for unknown or verified addresses and throttled to one send per 60 seconds |

Verification tokens are HS256 JWTs valid for 24 hours and are sent as links in
email. Each email also contains a 6-digit code valid for 10 minutes with at most
5 failed attempts. Verification is idempotent. Invalid, expired, or mismatched
tokens or codes return `400` with a generic message. Resend requests for unknown
or already verified addresses still return `202` to avoid enumeration.

### Organizations

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/orgs` | 201 | `{"name"}`; creates owner membership |
| GET | `/orgs` | 200 | Orgs where the actor has a role |
| GET | `/orgs/{orgID}?orgId={orgID}` | 200 | Any org role |
| PATCH | `/orgs/{orgID}?orgId={orgID}` | 200 | Requires `org.admin` or the owner wildcard; `{"name"}` |

Path-scoped org routes repeat `orgId` in the query and it must match the path.
`RequireOrgAccess` reads the tenant from the query, the role from the token
`orgs` map, and the required unit permission from `routes.yml`.

No org delete.

### Memberships

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/orgs/{orgID}/members?orgId={orgID}` | 201 | Requires `member.admin` or the owner wildcard; `{"userId","roles":{"expense":"editor"}}`; omitted roles use the legacy editor map |
| GET | `/orgs/{orgID}/members?orgId={orgID}` | 200 | Owner and `member.viewer`/`member.admin`; paginated |
| PATCH | `/orgs/{orgID}/members/{memberUserID}?orgId={orgID}` | 204 | Requires `member.admin`; replaces `{"roles":{"project":"viewer","expense":"editor"}}`; owner cannot change |
| DELETE | `/orgs/{orgID}/members/{memberUserID}?orgId={orgID}` | 204 | Requires `member.admin`; owner cannot be removed |

Membership responses include `owner: true` for the organization owner and a
`roles` map for everyone else. `"*"` is never accepted or stored. `member.admin`
is a high-trust role: it can manage membership and assign feature roles. The
owner account cannot be added, removed, or have its roles changed.

### Sharing

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/orgs/{orgID}/shares?orgId={orgID}` | 201 | `shares.create`; returns the raw token once |
| GET | `/orgs/{orgID}/shares?orgId={orgID}` | 200 | `shares.list`; no tokens |
| GET | `/orgs/{orgID}/shares/status?orgId={orgID}&resourceType=&resourceIds=` | 200 | `shares.status`; active links per resource |
| GET | `/orgs/{orgID}/credentials?orgId={orgID}&kind=&resourceType=&resourceIds=&active=` | 200 | `shares.credentials`; filter links and future API keys |
| DELETE | `/orgs/{orgID}/shares/{shareID}?orgId={orgID}` | 204 | `shares.revoke` |

Share access uses the existing routes:

```http
Authorization: Share <opaque-token>
```

The credential is hashed at rest, expiring, and revocable. Scopes are derived
from the share resource:

| Resource | Scopes |
| --- | --- |
| org | org, project list/get, expense list/summary/get |
| project | org, project get, project expense list/summary/get |
| expense | org, project get, expense get |

Object scope is enforced in controllers:

- project share forces `ProjectIDs` for list and summary;
- expense share rejects any expense ID other than its own;
- project and expense shares reject other projects;
- share responses omit `userId`.

### Links

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/orgs/{orgID}/links?orgId={orgID}` | 201 | `links.create`; `{"url","slug","title","tags"}` |
| GET | `/orgs/{orgID}/links?orgId={orgID}` | 200 | `links.list`; paginated; `q` alias prefix, `tags` any match |
| GET | `/orgs/{orgID}/links/{linkID}?orgId={orgID}` | 200 | `links.get` |
| PATCH | `/orgs/{orgID}/links/{linkID}?orgId={orgID}` | 200 | `links.update`; `url`, `title`, `tags`, `disabled` |
| DELETE | `/orgs/{orgID}/links/{linkID}?orgId={orgID}` | 204 | `links.delete`; tombstone |
| GET | `/s/{slug}` | 302 | Public redirect; no identity |

Links are publicly readable by slug, so the alias namespace is global rather
than per organization. An omitted alias is generated from a 32-character
lowercase alphabet with seven characters. Custom aliases are 3–32 characters,
lowercase, and immutable. Only `http` and `https` destinations are accepted.

The redirect response sets `Cache-Control: no-store` so edits and disables take
effect immediately, and `X-Robots-Tag: noindex`. Missing, disabled, and deleted
links return `404 {"error":"link not found"}`. There is no click tracking and no
write on redirect.

Delete is a tombstone (`deletedAt`) rather than a hard delete. The slug remains
reserved, which prevents a deleted alias from being recreated for a different
destination. Disable (`disabledAt`) is the reversible action.

The `link` feature owns management:

```yaml
link:
  viewer: [links.list, links.get]
  editor: [links.list, links.get, links.create, links.update]
  admin: [links.list, links.get, links.create, links.update, links.delete]
```

### Projects

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/projects?orgId={orgID}` | 201 | `{"name","currency","tags"}` |
| GET | `/projects?orgId={orgID}` | 200 | Paginated list; `tags` filter uses any match |
| GET | `/projects/{projectID}?orgId={orgID}` | 200 | |
| PATCH | `/projects/{projectID}?orgId={orgID}` | 200 | Partial `{"name"}` and/or `{"tags"}` |

Project `currency` is immutable after creation. Expenses inherit it. Project
tags are trimmed, deduplicated, sorted, and capped at 20 items with a 32-byte
limit per tag. `tags=food,travel` returns projects matching either tag; expense
tag filtering remains all-match.

No project delete.

### Expenses

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| POST | `/expenses?orgId={orgID}` | 201 | Create |
| GET | `/expenses?orgId={orgID}` | 200 | Filters: `projectId`, `projectIds`, `tags`, `tag`, `from`, `to`; returns one page |
| GET | `/expenses/summary?orgId={orgID}` | 200 | Same filters; totals grouped by project and currency |
| GET | `/expenses/{expenseID}?orgId={orgID}` | 200 | |
| PATCH | `/expenses/{expenseID}?orgId={orgID}` | 200 | Partial update |
| DELETE | `/expenses/{expenseID}?orgId={orgID}` | 204 | |

Expense `type` is `debit` for money out or `credit` for money in. It defaults to
debit when omitted on create, and records without a type are read and aggregated
as debit. Expense `amount` is a positive integer in minor units; the type carries
direction. `currency` is inherited from the project and cannot be changed through
the expense API. Tags are trimmed, deduplicated, sorted, and capped at 20 items.
`spentAt` is required.

`tags=food,travel` (or repeated `tags` parameters) requires every tag to match.
`tag` remains as a single-tag fallback. `from` and `to` are inclusive RFC 3339
bounds. `projectId` selects one project; `projectIds` selects up to 100 projects
as a comma-separated list or repeated parameter. Supplying both is a `400`. The
list response contains one page and the next offset:

```json
{
  "items": [],
  "nextOffset": 20
}
```

`nextOffset` is `null` on the last page. List requests do not compute totals.

`GET /expenses/summary` accepts the same filters and returns one totals row per
project and currency:

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

`totalDebit` and `totalCredit` are in minor units. `netTotal` is
`totalCredit - totalDebit`. Amounts are never added across currencies.

### Errors

| Status | Meaning |
| --- | --- |
| 400 | Validation error, malformed JSON, missing or mismatched `orgId`, invalid verification token |
| 401 | Missing, invalid, or expired access or refresh token; invalid login credentials |
| 403 | Email not verified, no role for the org, or the role lacks the route permission |
| 404 | Resource missing or malformed ObjectID |
| 405 | Method not allowed; includes `Allow` |
| 409 | Duplicate email/membership, owner removal, or conflict |
| 502 | Verification email delivery failed |
| 500 | Unexpected error; details are logged, not returned |

Invalid ObjectIDs are treated as not found rather than as a separate error.

## Identity and tenancy

Protected routes authenticate with `Authorization: Bearer <token>`, falling back
to the `pico_auth` cookie:

- `Authenticate` accepts either an access JWT or `Authorization: Share
  <opaque-token>`. A JWT stores its `sub` and `orgs` feature role lists; a share
  stores route scopes and resource scope from the `credentials` collection.
- `RequireAuth` accepts only a JWT and is used on user-scoped routes. A share
  credential cannot reach `/users` or `/auth` routes.
- `RequireOrgAccess` reads the tenant from `?orgId=`. It rejects a path/query
  mismatch with `400`, a missing tenant with `400`, and an organization absent
  from the token with `403`.
- The matched chi route pattern is looked up in `routes.yml`; a missing or
  non-org policy fails closed with `500`. The route key is the unit permission.
- `"*"` grants every defined permission for that organization. Otherwise any
  feature role in the token may grant the route permission through
  `features.yml`, or a share scope may grant it directly. Share requests also
  check the resource scope.

Authentication and authorization are declarative and in-memory:

```text
token sub + token orgs[orgId] feature roles + route pattern
+ routes.yml permission + features.yml grants
= allow or deny
```

Feature-management permissions are enforced by middleware: `org.admin` can
update organization settings, and `member.admin` can add, update, or remove
members. Services still protect the owner account: the owner cannot be added as
a member, removed, or have their roles changed. Creating an org inserts the org
and then the owner membership. If the membership insert fails, the org is
deleted as a best-effort cleanup.

The owner role is never declared in `features.yml` and `"*"` is never accepted
from a membership API. The policy is embedded from `internal/authz/resources`,
validated at startup, and checked against the chi router by `test/routes`.

## Email verification

- `POST /users` creates the user with `emailVerified: false`, signs a JWT, and
  sends it over SMTP. The link points at
  `{PUBLIC_BASE_URL}/verify-email#token=...`, where `PUBLIC_BASE_URL` is the UI
  origin. The UI reads the fragment and calls `POST /auth/verify-email` with the
  token in the JSON body.
- The same email contains a 6-digit code. `POST /auth/verify-email/otp` accepts
  `{"email":"...","code":"..."}` and marks the email verified.
- Codes expire after 10 minutes and allow at most 5 failed attempts. Only an
  HMAC-SHA256 hash is stored, keyed with `JWT_SECRET`.
- Tokens use HS256 with `JWT_SECRET` (at least 32 bytes). Claims are `sub`
  (user ID), `email`, `purpose=email_verification`, issuer, audience, `iat`, and
  `exp`. Tokens expire after 24 hours.
- Verification loads the user, compares the token or code with the stored
  state, and marks the user verified. Replaying a valid link is idempotent.
- Changing the email address clears verification and sends a new link and code.
  If delivery fails, the new address remains stored and unverified; resend can
  retry.
- Signup deletes the new user as a best-effort cleanup when the email cannot be
  sent, so the caller can retry.
- Resend is throttled to one send per 60 seconds and always returns `202` with
  `Retry-After: 60` and `{"status":"sent","retryAfterSeconds":60}` for unknown
  or verified addresses to prevent enumeration. It rotates the code.
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
orgs:         {userId:1, createdAt:-1, _id:-1}
memberships:  {orgId:1, userId:1} unique
              {userId:1, createdAt:-1, _id:-1}
              {orgId:1, createdAt:-1, _id:-1}
projects:     {orgId:1, createdAt:-1, _id:-1}
              {orgId:1, tags:1}
links:        {slug:1} unique
              {orgId:1, createdAt:-1, _id:-1}
              {orgId:1, tags:1}
              {orgId:1, slug:1}
credentials:  {tokenHash:1} unique
              {orgId:1, kind:1, resourceType:1, resourceId:1, revokedAt:1, expiresAt:1}
              {orgId:1, createdAt:-1, _id:-1}
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

### Stateless tokens with declarative org authorization

Login issues a 30-minute HS256 access JWT and a 30-day refresh token, returns
both in the response body, and sets `pico_auth` and `pico_refresh` HttpOnly
cookie fallbacks. Browser clients store both in `localStorage` and send the
access token as `Authorization: Bearer`. A `401` triggers one refresh attempt
against `POST /auth/refresh`, which re-queries memberships before reissuing the
pair. Token validation and authorization are local, so protected requests with
a valid token do not query users or memberships. Tokens cannot be revoked before
expiry; see Known debt.

The actor is always the access-token `sub`. The tenant is `?orgId=`, and on
`/orgs/{orgID}` routes it must match the path. `RequireOrgAccess` looks up the
matched chi route pattern in `internal/authz/resources/routes.yml`, resolves the
unit permission (`expenses.list`, `expenses.create`, and so on), and checks the
organization's feature roles against `features.yml`. The route YAML is embedded,
validated at startup, and covered by a router-walk test. There is no `?userId=`
fallback and no global user directory.

The organization owner is implicit. Token issuance writes `["*"]` for an
organization whose `orgs.userId` matches the subject; `["*"]` grants every defined
permission. It is never declared in `features.yml` and never stored on a
membership. This keeps the owner current when new permissions are added and
prevents an owner lockout caused by a forgotten role entry.

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

### Page reads and expense summaries are separate

Expense list requests return one page and `nextOffset`; they do not run an
aggregate. Totals are computed by `GET /expenses/summary` with the same match
filters and no pagination, grouped by project and currency. Clients never sum
the current page, so the displayed total does not change when the user
paginates, and a page request never scans the full filtered set. The summary
uses a `$group` aggregation for count, debit amount, and credit amount;
`netTotal` is computed as credit minus debit. An empty match returns no groups.
A legacy document with no `type` is counted as debit through `$ifNull`.

## Known debt

- **Tokens cannot be revoked.** Logout clears the client tokens and cookies, but
  a copied access token stays valid for up to 30 minutes and a copied refresh
  token for up to 30 days. Password changes do not invalidate existing tokens.
- **Refresh tokens are stateless and replayable.** Rotation returns a new token
  but does not invalidate the old one. Add token-version or session storage if
  immediate revocation is required.
- **Share tokens are opaque and stored hashed.** They are revocable, unlike
  access tokens. There is no last-used update on every request yet.
- **Share status is direct-only.** A project or org share does not mark its
  children as shared in `/shares/status`.
- **Short links are open redirects.** Creation requires an organization role,
  but there is no destination denylist, abuse reporting, or per-user creation
  rate limit. Add those before exposing creation to untrusted users at scale.
- **Access tokens carry at most 20 organization role lists.** `RolesForUser`
  caps the `orgs` map so the token stays small. A user with more memberships
  loses access to the oldest organizations until the cap is raised.
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
- Tag-filtered project and expense lists use multikey tag indexes that do not
  include the sort field. `explain` should confirm the plan before adding
  sort-aware compound indexes at larger data volumes.
- `GET /expenses/summary` groups totals by project and currency. Callers must
  not add the rows together across currencies.
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
  its prefix. Wrap org-scoped routes with `middleware.RequireOrgAccess` after
  `middleware.RequireAuth`, user-scoped routes with `middleware.RequireAuth`,
  and add the route to `internal/authz/resources/routes.yml` (plus
  `features.yml` for org routes). `test/routes` fails when the router and policy
  drift.
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
