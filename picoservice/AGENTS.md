# picoservice agent instructions

## Project snapshot

- Go 1.27, chi v5, a single AWS Lambda behind a public Function URL.
- No API Gateway. CORS is handled by the shared `middleware.CORS` layer in the
  chi router; the Function URL has no CORS configuration.
- MongoDB Atlas via the official `go.mongodb.org/mongo-driver/v2` driver.
- MongoDB-backed APIs for users, organizations, memberships, projects, and
  expenses.
- Projects own a currency; expenses inherit it at creation. Projects also carry
  up to 20 tags (32 bytes each) and can be filtered by any matching tag.
- Organizations carry up to 20 tags (32 bytes each), filter with any-match
  semantics, and allow owner tag edits. Organization tag filtering applies within
  the 100-membership list page.
- Expenses are debits (money out) or credits (money in). Amounts stay positive;
  `type` carries direction. Missing types are read and aggregated as debit.
- Expense lists support `projectId`, all-match `tags` (legacy single `tag`),
  inclusive `from`/`to`, and pagination. Responses include `totalCount`,
  `totalDebit`, `totalCredit`, and `netTotal` over the full filtered set;
  `netTotal` is `totalCredit - totalDebit`, and pagination affects only `items`.
- pico-ui serves a public `/api-docs` guide mirroring this API contract. Keep
  API changes reflected in `../pico-ui/src/features/api-docs/content.ts`.
- Health API: `GET /ping` returns `200 {"message":"pong"}`; `HEAD /ping`
  returns an empty `200`. `GET /ready` pings MongoDB and returns
  `200 {"status":"ok"}` or `503 {"error":"database unavailable"}`.
- Org-scoped routes require `orgId` and a valid membership. The actor comes from
  the `pico_auth` JWT cookie. Path-scoped org and member routes repeat `orgId`
  in the query.
- Protected routes authenticate with a stateless HS256 access JWT stored in an
  HttpOnly cookie. Login requires a verified email. The `userId` query parameter
  remains as a temporary, spoofable fallback until the UI migrates.
- Passwords are bcrypt-hashed and never returned. No user or org delete endpoints.
- Signup creates an unverified user and emails a signed JWT verification link.
  The link opens the UI, which calls `POST /auth/verify-email`; resend is
  throttled and generic.
- `JWT_SECRET`, `PUBLIC_BASE_URL`, `EMAIL_FROM`, and `SMTP_ADDR` are required.
  `PUBLIC_BASE_URL` is the UI origin and the default CORS origin.
  `CORS_ALLOWED_ORIGINS` optionally overrides the CORS origin list.
  `SMTP_USERNAME` and `SMTP_PASSWORD` are optional and set together.
- Build with SAM (`template.yaml`); run locally with `make run`.
- `MONGODB_URI` and `MONGODB_DATABASE` are required.
- Local settings live in `.env` (ignored); `sample.env` is committed. Make targets
  for `run`, `indexes`, `test-integration`, and `invoke` load `.env`; `deploy`
  stays explicit.
- Unique indexes are created with `make indexes`, not during Lambda startup.

## Read first

- `docs/architecture.md` — request flow, package boundaries, data model, API, decisions, known debt.
- `docs/engineering-standards.md` — clean code, DRY, modularity, testing, and verification rules.

## Commands

| Command | Purpose |
| --- | --- |
| `make run` | Run the local HTTP server on `:8080` (loads `.env`) |
| `make test` | Run Go tests only (integration tests skip) |
| `make test-integration` | Run MongoDB integration tests (loads `.env`; requires `MONGODB_TEST_URI`) |
| `make check` | Check formatting, run `go vet`, and run tests |
| `make indexes` | Create MongoDB indexes (loads `.env`; requires `MONGODB_URI` and `MONGODB_DATABASE`) |
| `make build` | Run `sam build` |
| `make invoke` | Invoke the Lambda locally with `events/function-url.json` (Docker and `.env` required) |
| `make deploy` | Deploy the SAM stack (explicit env; does not load `.env`) |
| `make ship` | Deploy through `scripts/deploy.sh` using `deploy.env` |
| `make url` | Print the deployed Function URL |
| `sam validate --lint` | Validate template changes |

## Non-negotiable rules

- **Boundaries:** Lambda/AWS event types belong only in `internal/lambdahttp`. MongoDB driver types belong only in `internal/dao/mongodb` in production code; integration tests may use the driver to isolate and drop test databases. Transport middleware belongs only in `internal/middleware`. SMTP and message construction belong only in `internal/email`; services consume the `services.Mailer` interface. Controllers must not import `events`, `lambda`, AWS SDKs, `internal/dao`, or the MongoDB driver.
- **Dependency direction:** `main`/`cmd` → `internal/app` → `routes`/`services`/`dao/mongodb`/`email`; `routes` → `controllers`, `middleware`, `httpx`; `controllers` → `services`, `middleware`, `entities`, `httpx`, `validation`; `services` → `dao`, `entities`, `validation`, stdlib, JWT; `dao` → `entities`; `dao/mongodb` → `dao`, `entities`, MongoDB driver; `email` → stdlib; `middleware` → `httpx`, chi; `lambdahttp` → `httpx`; `entities`, `httpx`, and `validation` import stdlib only. Controllers depend on small consumer-side service interfaces.
- **Thin controllers:** controllers parse/validate HTTP input and write responses. Business logic lives in services. DTOs and response mapping stay in `internal/controllers`; do not query MongoDB from controllers.
- **Tenancy:** org-scoped routes must use `middleware.RequireMember`; user-scoped routes that need an actor must use `middleware.RequireAuth`. Owner-only rules live in services and compare the actor to `entities.Org.UserID`. Never describe query parameters as authentication.
- **Passwords:** hash with bcrypt; never store plaintext, log passwords, or return hashes. Password changes verify the current password. Email verification tokens are not access tokens and must never authorize API calls.
- **JSON contract:** all responses are JSON. Use the shared helpers in `internal/httpx` and the error shape `{"error":"..."}`. HEAD responses carry no body. `204` responses have no body.
- **DRY:** keep constants, response helpers, route registration, validation, and config in one place. The domain subrouters in `internal/routes` are the source for registration, middleware, and each path's `Allow` header; `router.go` mounts them under their prefixes. Apply the rule of three; do not abstract prematurely.
- **CORS:** use the shared `middleware.CORS` layer wired in `routes.New`.
  `CORS_ALLOWED_ORIGINS` defaults to `PUBLIC_BASE_URL`; `*` reflects any
  non-null origin and is development-only. Do not also configure Function URL
  CORS; duplicate `Access-Control-*` headers break browsers.
- **Architecture:** do not add API Gateway, replace chi, add an ODM/generic repository, or add a Lambda proxy/framework library without explicit approval.
- **State:** no global mutable state, no `init()`, no hidden I/O. Wire dependencies in `internal/app`; create one MongoDB client per process and never per request.
- **Configuration:** application values belong in `.env` (ignored), with `sample.env` as the committed template. `make run`, `indexes`, `test-integration`, and `invoke` load `.env`; `make deploy` must not. AWS deployment settings belong in `deploy.env` (ignored), with `sample.deploy.env` committed; `make ship`/`scripts/deploy.sh` load `.env` for application values and `deploy.env` for AWS settings. Do not put application secrets in `deploy.env`.
- **Database:** every call takes a `context.Context` with a timeout; log driver errors with `request_id` and never return them to clients. DAO contracts live in `internal/dao`; implementations live in `internal/dao/mongodb`. Add indexes in `internal/dao/mongodb/indexes.go`.
- **Errors:** wrap with `%w`, log once at the boundary, never silently ignore. No `log.Fatal`/`os.Exit` outside `main`.
- **Logging:** use `log/slog`; never log secrets, tokens, connection strings, passwords, or full request bodies.
- **Dependencies:** stdlib, chi, `aws-lambda-go`, `mongo-driver/v2`, `golang.org/x/crypto` (bcrypt), and `github.com/golang-jwt/jwt/v5` (email verification and access tokens) are approved. Justify new dependencies and verify they are maintained.
- **Tests:** every behavior change and bug fix needs a test. Tests live under `test/` and use only exported APIs. Prefer table-driven tests and `httptest`. Integration tests must skip unless `MONGODB_TEST_URI` is set.
- **Docs:** update `README.md`, `docs/architecture.md`, `docs/engineering-standards.md`, and `AGENTS.md` when the API or architecture changes.
- **Known debt:** if you touch an area listed in `docs/architecture.md` under Known debt, fix it or explain why not.

## Adding a route

1. Add or extend the domain record in `internal/entities`.
2. Add request/response DTOs and the controller method in
   `internal/controllers`.
3. Add the service input and business logic in `internal/services`; keep the
   controller thin.
4. Add or extend the DAO contract in `internal/dao` and its MongoDB
   implementation in `internal/dao/mongodb`.
5. Put email delivery behind `services.Mailer` and implement message/SMTP work
   in `internal/email`.
6. Add behavior tests in `test/controllers`, `test/services`, and
   `test/dao/mongodb` using `httptest` or integration fixtures as appropriate.
7. Register the route in the matching domain subrouter in `internal/routes`
   (`routes_users.go`, `routes_orgs.go`, and so on), including
   `middleware.RequireAuth` or `middleware.RequireMember` where appropriate. The
   per-route 405 `Allow` header is derived automatically.
8. Update the route contract in `README.md` and `docs/architecture.md`.
9. Add indexes in `internal/dao/mongodb/indexes.go` if the route introduces a
   new query pattern or uniqueness constraint.
10. Run `make check`; run `sam validate --lint` if the template changed.

## Definition of done

- [ ] Tests added or updated
- [ ] `MONGODB_TEST_URI` integration tests pass when MongoDB behavior changed and credentials are available
- [ ] `make indexes` updated/documented if indexes changed
- [ ] `make check` passes
- [ ] `sam validate --lint` passes if `template.yaml` changed
- [ ] Docs updated if the API or architecture changed
- [ ] No `.aws-sam/` artifacts, secrets, or unrelated formatting changes
