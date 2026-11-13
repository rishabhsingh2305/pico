# Engineering standards

These rules apply to all code in this repository. Keep them short and update
them when the service grows.

## Principles

- **KISS:** choose the simplest design that satisfies the current requirement.
- **YAGNI:** do not build for hypothetical routes, databases, or frameworks.
- **DRY, with the rule of three:** do not duplicate logic or constants.
  Abstract after the third real repetition, not the first.
- **Single responsibility:** each package, file, and function should have one
  reason to change.
- **Explicit over clever:** prefer readable code over compact code.
- **Composition over inheritance:** use small interfaces and plain functions.

## Go conventions

- Format with `gofmt`; `make check` verifies formatting.
- Follow [Effective Go](https://go.dev/doc/effective_go) and
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Package names are lowercase, short, and descriptive. No `utils`, `common`, or
  `helpers` dumping grounds.
- Exported identifiers have doc comments. Use comments to explain *why*, not
  *what*.
- Keep functions small. Use guard clauses and early returns over nested
  conditionals.
- Avoid boolean parameters that change behavior; use separate functions or an
  options struct instead.
- Accept interfaces, return concrete types. Define interfaces at the consumer.
- Use `context.Context` as the first parameter for I/O and pass it through.
- No package-level mutable state, no `init()`, and no hidden I/O.
- Use `http.Status*` constants and named constants for repeated strings and
  numbers.

## Boundaries and modularity

- Follow the dependency direction in `docs/architecture.md`.
- AWS/Lambda types stay in `internal/lambdahttp`. MongoDB driver types stay in
  `internal/dao/mongodb`; controllers depend on small service interfaces, not
  on the driver.
- Controllers are HTTP-only: no business logic, AWS SDK calls, or direct
  database access. HTTP request/response DTOs and response mapping live in
  `internal/controllers`.
- Services (`internal/services`) own validation and business rules. They consume
  DAO contracts in `internal/dao` and define narrow lookup interfaces for
  cross-domain dependencies.
- `internal/entities` contains stdlib-only domain records and sentinel errors.
- `internal/dao` contains per-domain persistence contracts and query/update
  structs; `internal/dao/mongodb` contains the MongoDB implementations.
- `internal/email` owns SMTP delivery and message construction. Services consume
  the `services.Mailer` interface; controllers never send email.
- `internal/middleware` owns HTTP middleware: request ID propagation,
  structured request logging, panic recovery, and query identity/membership
  checks. It defines its own lookup interfaces and may import `internal/httpx`
  and chi middleware only.
- `lambdahttp` adapts transport only; it must not know about routes or
  controllers.
- `internal/app` is the shared composition root for the Lambda and local
  server. `main` and `cmd` wire the MongoDB client and application handler only.
- Add a package only when it has a clear responsibility. Do not create generic
  layers for a single route.
- Prefer standard `net/http` behavior over framework-specific abstractions.

## DRY and single sources of truth

- One route tree: `internal/routes/router.go` mounts one subrouter per domain
  (`routes_users.go`, `routes_orgs.go`, and so on). The main router defines the
  prefixes; domain files define relative paths and their per-route identity
  middleware. A flat probe router built from the route tree supplies each path's
  `Allow` header. Update docs when routes change.
- One JSON response helper and one error envelope in `internal/httpx`. Do not
  re-implement `Content-Type`, status writing, or `{"error":"..."}` in multiple
  packages.
- One constant for `X-Request-Id`; one constant for the JSON content type. Both
  live in `internal/httpx`.
- Shared validation errors and pagination rules live in `internal/validation`.
  Services should not duplicate the validation error shape or limit/offset
  normalization.
- Configuration values live in `template.yaml` or `samconfig.toml`; code reads
  environment variables and keeps their names in one place. Local development
  uses `.env` (ignored) via the Makefile; `sample.env` is the committed template.
  Do not duplicate environment values in code.
- Tests may share helpers, but do not build a test framework for one test.
- Do not DRY unrelated code just because it looks similar. Duplication is often
  cheaper than a bad abstraction.

## Errors, logging, and observability

- Return errors; do not panic for expected failures.
- Wrap errors with `%w` and context:
  `fmt.Errorf("load config: %w", err)`.
- Log once, at the boundary (`main`, adapter, or middleware). Do not log and
  return the same error at every layer.
- Use `log/slog` with structured key/value pairs. No `fmt.Println` for
  operational logs.
- Include `request_id` on request-scoped logs.
- Never log secrets, tokens, full request bodies, or personal data.
- Use `context` cancellation and timeouts for downstream I/O.
- Do not use `log.Fatal` or `os.Exit` outside `main`.

## HTTP and API conventions

- JSON by default; set `Content-Type: application/json; charset=utf-8`.
- Error body: `{"error":"message"}`. Do not leak internal errors.
- Use correct status codes: 200, 201, 202, 204, 400, 403, 404, 405, 409, 502,
  500.
- 404 and 405 responses are JSON, not plain text.
- Set `Allow` on 405 responses; derive it from the router tree, never from a
  global method list.
- HEAD responses carry headers and no body.
- List endpoints return one page and pagination metadata. They must not run an
  unbounded aggregate over the same filter. Expense totals live on
  `GET /expenses/summary`; they cover the full filtered set but are never
  computed by the list query. Never make clients derive totals from one page.
- Aggregate endpoints group server-side. Clients must not issue one totals
  request per project, category, or row. Amounts are grouped by currency and
  never added across currencies. Expense totals are split into debit and credit
  sums, with net computed as credit minus debit.
- Expense amounts are positive; `type` carries debit/credit direction. Missing
  types default to debit on create, read, and aggregation.
- Keep filter semantics explicit in the API contract and docs. Project tags use
  any-match filtering; expense tags use all-match filtering.
- If a route must return a non-JSON body, set the correct `Content-Type`; the
  adapter base64 encodes binary bodies at the transport boundary.
- CORS is configured only on the Function URL. No CORS middleware in the router.
- Use the Lambda request ID when available; the local server uses chi's
  generated ID.
- Do not add API Gateway or change Function URL auth/CORS without explicit
  approval.

## Identity, tenancy, and passwords

- Protected routes authenticate with the `pico_auth` JWT cookie. The temporary
  `userId` query fallback is not secure; never describe it as authentication.
- Email verification proves ownership of an email address only. Verification
  JWTs must never be accepted as access tokens or used to authorize requests.
- Every org-scoped route must use `middleware.RequireMember`. User-scoped routes
  that need an actor must use `middleware.RequireAuth`.
- The route tree, not individual controllers, decides which middleware wraps a
  route.
- Membership enforcement is not authorization by itself. Owner-only rules belong
  in services and compare the actor to `entities.Org.UserID`.
- Passwords are hashed with bcrypt. Never store plaintext, log passwords, or
  return hashes. Password changes must verify the current password.
- Bound passwords to 8–72 bytes because of bcrypt. Normalize emails to lowercase
  before enforcing the unique index.

## Database access

- One MongoDB client per process, created during startup and closed on
  shutdown. Never create a client per request.
- `mongo.Connect` is lazy: do not ping at startup. Connectivity checks belong in
  `/ready`.
- Every database call takes a `context.Context` and a timeout.
- Every list query must have an index that supports its equality filters and sort
  order. Range filters may share the same compound index when the sort follows
  the range field. Add or change indexes in
  `internal/dao/mongodb/indexes.go`, then run `make indexes`.
- Tag indexes are multikey. Verify with `explain("executionStats")` that a
  tag-filtered query does not fall back to a collection scan or an unnecessary
  blocking sort.
- List queries return one page. Aggregate queries live on separate summary
  endpoints; do not compute totals in the list path. Do not use `$facet` to
  combine an unbounded summary with a page.
- Offset pagination is acceptable for small, bounded lists. Use cursor
  pagination for deep or high-volume lists.
- Index changes are migrations: add a new index before deploying dependent code,
  verify it, then remove the old index in a separate change. Never reuse an index
  name with different keys.
- Controllers depend on small consumer-side service interfaces, not on the
  driver or `*mongo.Client`.
- DAO contracts live in `internal/dao`. Concrete stores live in
  `internal/dao/mongodb`. Do not add a generic `Repository[T]` or an ODM.
- Store `_id` values as ObjectIDs and references (`orgId`, `userId`,
  `projectId`) as ObjectID hex strings. Keep the representation consistent
  across collections.
- Unique indexes are correctness constraints. Add them in
  `internal/dao/mongodb/indexes.go`; do not create them during Lambda startup.
- User and membership uniqueness depends on `make indexes`. A deployment without
  the indexes can accept duplicates.
- Multi-document operations are not transactional. Document best-effort cleanup
  and avoid implying atomicity.
- Log database errors with the request ID and return a stable message; never
  expose driver errors to clients.
- Always close the client during shutdown and check the returned error.

## Testing

- Every behavior change and bug fix gets a test.
- Put tests under `test/`, mirroring `internal/` layers (`test/controllers`,
  `test/services`, `test/routes`, `test/dao/mongodb`, `test/email`); use only
  exported APIs.
- Prefer table-driven tests and `httptest` for HTTP behavior.
- Test behavior, not private implementation details.
- Cover success, 4xx, 5xx, and malformed-input paths.
- Tests must be deterministic: no sleeps, real network calls, or external
  services.
- Integration tests that need a real service must skip unless their environment
  variable is set (`MONGODB_TEST_URI` for MongoDB); they must not run in
  `make check`.
- Integration tests must isolate their data. The MongoDB store test clears its
  collections in `MONGODB_TEST_DATABASE` before and after it runs.
- Keep test helpers local until reused by multiple test files. Shared service
  doubles live in `test/fakes`.
- `make check` is the required verification command.

## Dependencies and configuration

- Stdlib first. chi, `aws-lambda-go`, `mongo-driver/v2`,
  `golang.org/x/crypto` (bcrypt), and `github.com/golang-jwt/jwt/v5` (email
  verification) are the approved dependencies.
- Do not add archived or unmaintained packages. Verify maintenance status before
  adding one.
- Pin versions in `go.mod` and run `go mod tidy`.
- New dependencies need a short justification in the change description.
- No secrets in code or config. Use environment variables or AWS-managed
  configuration.
- Never edit files under `.aws-sam/`; they are generated build artifacts.

## Change checklist

- [ ] The change is scoped to one concern.
- [ ] Names and package boundaries follow this document.
- [ ] Errors are wrapped and logged at the boundary.
- [ ] Tests cover success, 4xx, and 5xx paths.
- [ ] `make check` passes.
- [ ] `sam validate --lint` passes for template changes.
- [ ] `make indexes` is documented/updated for index changes.
- [ ] New or changed query patterns were checked with `explain` and are covered
      by an index.
- [ ] List handlers do not run unbounded aggregates, and clients do not issue
      one totals request per item.
- [ ] `README.md` and `docs/architecture.md` are updated for contract changes.
- [ ] No generated, secret, or unrelated files are included.

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
- [chi](https://github.com/go-chi/chi)
- [Lambda Function URLs](https://docs.aws.amazon.com/lambda/latest/dg/urls-invocation.html)
- [Lambda Go handlers](https://docs.aws.amazon.com/lambda/latest/dg/golang-handler.html)
