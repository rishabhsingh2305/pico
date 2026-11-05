.PHONY: build test test-integration fmt fmt-check vet check tidy run invoke deploy ship url indexes clean

# Load local development configuration for selected commands. Deployment stays
# explicit so local .env values cannot be deployed accidentally.
ENV_FILE ?= ./.env
LOAD_ENV = set -a; if [ -f "$(ENV_FILE)" ]; then . "$(ENV_FILE)"; fi; set +a;

build:
	sam build

test:
	go test ./...

test-integration:
	@$(LOAD_ENV) \
	test -n "$$MONGODB_TEST_URI" || (echo "MONGODB_TEST_URI is required"; exit 1); \
	go test ./test/dao/mongodb -run Integration -count=1

fmt:
	gofmt -w .

fmt-check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required for:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	go vet ./...

check: fmt-check vet test

tidy:
	go mod tidy

run:
	@$(LOAD_ENV) go run ./cmd/server

invoke:
	@$(LOAD_ENV) \
	test -n "$$MONGODB_URI" || (echo "MONGODB_URI is required"; exit 1); \
	test -n "$$JWT_SECRET" || (echo "JWT_SECRET is required"; exit 1); \
	test -n "$$PUBLIC_BASE_URL" || (echo "PUBLIC_BASE_URL is required"; exit 1); \
	test -n "$$EMAIL_FROM" || (echo "EMAIL_FROM is required"; exit 1); \
	test -n "$$SMTP_ADDR" || (echo "SMTP_ADDR is required"; exit 1); \
	params="MongoDBURI=$$MONGODB_URI JwtSecret=$$JWT_SECRET PublicBaseURL=$$PUBLIC_BASE_URL EmailFrom=$$EMAIL_FROM SmtpAddr=$$SMTP_ADDR"; \
	if [ -n "$$SMTP_USERNAME" ]; then params="$$params SmtpUsername=$$SMTP_USERNAME SmtpPassword=$$SMTP_PASSWORD"; fi; \
	sam local invoke ApiFunction -e events/function-url.json --parameter-overrides "$$params"

deploy:
	@test -n "$$MONGODB_URI" || (echo "MONGODB_URI is required"; exit 1)
	@test -n "$$JWT_SECRET" || (echo "JWT_SECRET is required"; exit 1)
	@test -n "$$PUBLIC_BASE_URL" || (echo "PUBLIC_BASE_URL is required"; exit 1)
	@test -n "$$EMAIL_FROM" || (echo "EMAIL_FROM is required"; exit 1)
	@test -n "$$SMTP_ADDR" || (echo "SMTP_ADDR is required"; exit 1)
	@params="MongoDBURI=$$MONGODB_URI JwtSecret=$$JWT_SECRET PublicBaseURL=$$PUBLIC_BASE_URL EmailFrom=$$EMAIL_FROM SmtpAddr=$$SMTP_ADDR"; \
	if [ -n "$$SMTP_USERNAME" ]; then params="$$params SmtpUsername=$$SMTP_USERNAME SmtpPassword=$$SMTP_PASSWORD"; fi; \
	sam deploy --parameter-overrides "$$params"

ship:
	./scripts/deploy.sh $(ARGS)

url:
	aws cloudformation describe-stacks --stack-name picoservice --query "Stacks[0].Outputs[?OutputKey=='FunctionUrl'].OutputValue" --output text

indexes:
	@$(LOAD_ENV) \
	test -n "$$MONGODB_URI" || (echo "MONGODB_URI is required"; exit 1); \
	test -n "$$MONGODB_DATABASE" || (echo "MONGODB_DATABASE is required"; exit 1); \
	go run ./cmd/indexes

clean:
	rm -rf .aws-sam
