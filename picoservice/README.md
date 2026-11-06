# picoservice

A small Go REST API running as a single AWS Lambda function behind a public
Lambda Function URL. Routing is handled by
[chi](https://github.com/go-chi/chi), with a small adapter that translates
Function URL events into standard `net/http` requests. MongoDB Atlas stores
users, organizations, memberships, projects, and expenses.

## Endpoints

All request and response bodies are JSON. List responses accept `limit` and
`offset` query parameters (default `20`, maximum `100`). Most list responses use
`{"items":[...]}`; expense lists also return filtered totals. Read endpoints
also support `HEAD`, except the email verification action, which is `POST` only.

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
| `POST` | `/auth/login` | Sign in with `{"email","password"}`; sets an HttpOnly `pico_auth` cookie and returns `{"user","expiresAt"}` |
| `POST` | `/auth/logout` | Clear the auth cookie (`204`) |
| `GET` | `/auth/session` | Return the current user and token expiry, or `401` |

Access tokens are HS256 JWTs valid for 7 days and carry `sub`, `iss`,
`aud=picoservice-api`, and `type=access`. Login requires a verified email:
wrong credentials return `401 invalid email or password`, and unverified
accounts return `403 email not verified`. Tokens are stateless; logout clears
the cookie but does not revoke an already-issued token.

### Users

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/users` | Register with `{"name","email","password"}`; sends a verification email |
| `GET` | `/users?userId=actor` | List users |
| `GET` | `/users/{userID}?userId=actor` | Get a user |
| `PATCH` | `/users/{userID}?userId=actor` | Change `name` and/or `email`; self only. Changing the email resets verification |
| `POST` | `/users/{userID}/password?userId=actor` | Change password with `currentPassword` and `newPassword`; self only |

Protected routes use the auth cookie. Until the UI migrates, requests without a
cookie may still use the temporary `userId` query parameter.

Passwords are bcrypt-hashed. Password hashes are never returned. There is no
`DELETE /users` yet. User responses include `emailVerified`.

### Email verification

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/auth/verify-email` | Verify with `{"token":"..."}` from the UI link |
| `POST` | `/auth/verify-email/resend` | Resend with `{"email":"..."}`; returns `202` for valid input and is throttled to once per 60 seconds |

Verification links contain a signed JWT valid for 24 hours and return
`{"status":"verified","userId":"...","email":"..."}`. Signup creates the user
as unverified. Changing the
email marks the account unverified and sends a new link. Resend responses are
deliberately generic so they do not reveal whether an email is registered.

### Organizations

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/orgs?userId=actor` | Create `{"name","tags"}`; the actor becomes owner and a membership is created |
| `GET` | `/orgs?userId=actor&tags=food,travel` | List orgs the actor belongs to; filters: `tags` (any match) |
| `GET` | `/orgs/{orgID}?orgId=actor&userId=actor` | Get an org; members only |
| `PATCH` | `/orgs/{orgID}?orgId=actor&userId=actor` | Partial update of `name` and/or `tags`; owner only |

Organization tags are trimmed, deduplicated, sorted, and capped at 20 items with
a 32-byte limit per tag. `tags=food,travel` returns organizations matching
**any** of the requested tags. `tags: []` on PATCH clears all tags. Existing
organizations without a tags field read as an empty list.

Path-scoped org routes repeat `orgId` in the query because `RequireMember` reads
identity from query parameters.

There is no `DELETE /orgs`.

### Memberships

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/orgs/{orgID}/members?orgId=actor&userId=actor` | Add `{"userId":"..."}`; owner only |
| `GET` | `/orgs/{orgID}/members?orgId=actor&userId=actor` | List members; members only |
| `DELETE` | `/orgs/{orgID}/members/{memberUserID}?orgId=actor&userId=actor` | Remove a member; owner only and never the owner |

Membership roles are derived: the org's `userId` is the `owner`; everyone else
with a row is a `member`.

### Projects

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/projects?orgId=&userId=` | Create `{"name","currency","tags"}` |
| `GET` | `/projects?orgId=&userId=` | List projects; filters: `tags` (any match) |
| `GET` | `/projects/{projectID}?orgId=&userId=` | Get a project |
| `PATCH` | `/projects/{projectID}?orgId=&userId=` | Partial update of `name` and/or `tags` |

Project `currency` is a three-letter ISO code. Expenses inherit it and cannot
change it. Tags are trimmed, deduplicated, sorted, and capped at 20 items with a
32-byte limit per tag. `tags=food,travel` returns projects matching **any** of
the requested tags. This differs from expense tag filtering, which requires all
tags to match.

There is no project delete endpoint.

### Expenses

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/expenses?orgId=&userId=` | Create an expense |
| `GET` | `/expenses?orgId=&userId=` | List expenses; filters: `projectId`, `tags`, `tag`, `from`, `to` |
| `GET` | `/expenses/{expenseID}?orgId=&userId=` | Get an expense |
| `PATCH` | `/expenses/{expenseID}?orgId=&userId=` | Update fields |
| `DELETE` | `/expenses/{expenseID}?orgId=&userId=` | Delete an expense |

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
`from` and `to` are inclusive RFC 3339 bounds. Pagination applies only to the
returned `items`.

The list response includes totals over every expense matching the filters,
ignoring `limit` and `offset`:

```json
{
  "items": [],
  "totalCount": 42,
  "totalDebit": 100000,
  "totalCredit": 123456,
  "netTotal": 23456
}
```

`totalDebit` and `totalCredit` are sums in minor units. `netTotal` is
`totalCredit - totalDebit`. When `projectId` is set, the totals use the
project's currency. An organization-wide total across projects with different
currencies would need grouping by currency.

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

**Authentication uses a JWT cookie.** Protected routes read the `pico_auth`
cookie and trust its `sub` claim after verifying the signature and expiry; no
user lookup is performed for a valid token. Until the UI migrates, requests
without a cookie may still use the temporary `userId` query parameter, which
remains spoofable. Membership checks verify that the actor belongs to the org.
`GET /users` is still a global directory for authenticated users.

## Data model and indexes

```text
users
  _id, name, email (unique), passwordHash, emailVerified,
  emailVerifiedAt, verificationSentAt, createdAt, updatedAt

orgs
  _id, name, userId (owner), createdAt, updatedAt

memberships
  _id, orgId, userId, createdAt
  unique (orgId, userId)

projects
  _id, orgId, userId, name, currency, tags, createdAt, updatedAt

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
cp sample.env .env   # first time only
# Edit .env: MONGODB_URI, JWT_SECRET, SMTP settings, and so on.

make indexes
make run
```

Generate a local JWT secret with `openssl rand -hex 32`.

`make run`, `make indexes`, `make test-integration`, and `make invoke` load
`.env` automatically. `make deploy` intentionally does not; pass deployment
values explicitly. Exported environment variables still work for commands that
do not load `.env`.

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

# Create an org owned by that user.
curl -sS -X POST 'http://localhost:8080/orgs?userId=USER_ID' \
  -H 'Content-Type: application/json' \
  -d '{"name":"Acme"}'

# Create a project in the org.
curl -sS -X POST 'http://localhost:8080/projects?orgId=ORG_ID&userId=USER_ID' \
  -H 'Content-Type: application/json' \
  -d '{"name":"Launch","currency":"USD","tags":["travel","food"]}'

# List projects matching any of the tags.
curl -sS 'http://localhost:8080/projects?orgId=ORG_ID&userId=USER_ID&tags=travel,food'

# Record a debit (money out). Currency comes from the project.
curl -sS -X POST 'http://localhost:8080/expenses?orgId=ORG_ID&userId=USER_ID' \
  -H 'Content-Type: application/json' \
  -d '{"projectId":"PROJECT_ID","type":"debit","amount":1299,"description":"Lunch","tags":["food"],"spentAt":"2026-01-02T12:30:00Z"}'

# Record a credit (money in).
curl -sS -X POST 'http://localhost:8080/expenses?orgId=ORG_ID&userId=USER_ID' \
  -H 'Content-Type: application/json' \
  -d '{"projectId":"PROJECT_ID","type":"credit","amount":5000,"description":"Client payment","spentAt":"2026-01-03T09:00:00Z"}'

# List one project page and totals over every matching expense.
curl -sS 'http://localhost:8080/expenses?orgId=ORG_ID&userId=USER_ID&projectId=PROJECT_ID&tags=food,travel&from=2026-01-01T00%3A00%3A00Z&to=2026-01-31T23%3A59%3A59Z&limit=20&offset=0'
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
- loads `.env` for application settings and `deploy.env` for AWS settings
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

To keep deployment application values separate from local development, point
`APP_ENV_FILE` at a gitignored file:

```bash
APP_ENV_FILE=.env.production make ship
```

`deploy.env` is ignored and contains AWS deployment settings only. Application
secrets stay in the app env file and are never committed.

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

AWS deployment settings live in `deploy.env` (template: `sample.deploy.env`):
`AWS_PROFILE` or raw `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`,
`AWS_REGION`, optional `AWS_DEFAULT_REGION` (the script derives it from
`AWS_REGION` when omitted), optional `S3_BUCKET` for a dedicated SAM artifact
bucket (created automatically on first deploy), and optional `STACK_NAME`
(default `picoservice`). Application settings never go in `deploy.env`.

`make ship` loads `.env` for application values (override with
`APP_ENV_FILE`) and `deploy.env` for AWS settings. `make deploy` reads explicit
environment values only; local `make` targets read `.env`; see `sample.env`.

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
