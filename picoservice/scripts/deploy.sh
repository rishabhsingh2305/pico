#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

AWS_ENV_FILE="${DEPLOY_ENV_FILE:-deploy.env}"
APP_ENV_FILE="${APP_ENV_FILE:-.env}"
if [[ -n "${APP_ENV:-}" ]]; then
  SAMPLE_FILE="sample.${APP_ENV}.deploy.env"
else
  SAMPLE_FILE="sample.deploy.env"
fi
ASSUME_YES=0
SKIP_INDEXES=0
WATCH=0
CREATE_BUCKET=1

usage() {
  cat <<'EOF'
Usage: ./scripts/deploy.sh [options]

Options:
  -y, --yes           Skip the SAM changeset confirmation
      --skip-indexes  Do not run the MongoDB index command
      --no-create-bucket
                      Fail if S3_BUCKET is missing instead of creating it
      --watch         After deploying, run sam sync --watch
  -h, --help          Show this help

Environment:
  DEPLOY_ENV_FILE     AWS/deployment settings file, defaults to deploy.env
  APP_ENV_FILE        Application settings file, defaults to .env
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -y | --yes)
      ASSUME_YES=1
      shift
      ;;
    --skip-indexes)
      SKIP_INDEXES=1
      shift
      ;;
    --no-create-bucket)
      CREATE_BUCKET=0
      shift
      ;;
    --watch)
      WATCH=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

require_secure_file() {
  local file="$1"
  local sample="$2"

  if [[ ! -f "$file" ]]; then
    echo "Missing $file." >&2
    if [[ -n "$sample" ]]; then
      echo "Create it from the sample:" >&2
      echo "  cp $sample $file" >&2
      echo "  chmod 600 $file" >&2
    fi
    exit 1
  fi

  local perms
  perms="$(stat -c %a "$file" 2>/dev/null || stat -f %Lp "$file" 2>/dev/null || true)"
  if [[ "$perms" != "600" && "$perms" != "400" ]]; then
    echo "Refusing to continue: $file has permissions ${perms:-unknown}." >&2
    echo "Run: chmod 600 $file" >&2
    exit 1
  fi
}

require_secure_file "$APP_ENV_FILE" "sample.env"
require_secure_file "$AWS_ENV_FILE" "$SAMPLE_FILE"

# shellcheck disable=SC1090
set -a
source "$APP_ENV_FILE"
source "$AWS_ENV_FILE"
set +a

# SAM/boto3 reads AWS_DEFAULT_REGION, while the AWS CLI also accepts AWS_REGION.
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-$AWS_REGION}"

require_var() {
  local name="$1"
  local file="$2"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing $name in $file." >&2
    exit 1
  fi
}

for name in MONGODB_URI MONGODB_DATABASE JWT_SECRET PUBLIC_BASE_URL EMAIL_FROM SMTP_ADDR; do
  require_var "$name" "$APP_ENV_FILE"
done

require_var AWS_REGION "$AWS_ENV_FILE"
STACK_NAME="${STACK_NAME:-picoservice}"

if [[ "${#JWT_SECRET}" -lt 32 ]]; then
  echo "JWT_SECRET must be at least 32 bytes. Generate one with:" >&2
  echo "  openssl rand -hex 32" >&2
  exit 1
fi

if [[ -n "${SMTP_USERNAME:-}" && -z "${SMTP_PASSWORD:-}" ]] ||
  [[ -z "${SMTP_USERNAME:-}" && -n "${SMTP_PASSWORD:-}" ]]; then
  echo "Set both SMTP_USERNAME and SMTP_PASSWORD, or neither." >&2
  exit 1
fi

if [[ -z "${AWS_PROFILE:-}" && (-z "${AWS_ACCESS_KEY_ID:-}" || -z "${AWS_SECRET_ACCESS_KEY:-}") ]]; then
  echo "Set AWS_PROFILE, or AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, in $AWS_ENV_FILE." >&2
  exit 1
fi

if [[ "$MONGODB_URI" == *"USER:PASSWORD"* ]]; then
  echo "Fill in MONGODB_URI in $APP_ENV_FILE." >&2
  exit 1
fi

if [[ "$JWT_SECRET" == replace-with-* ]]; then
  echo "Fill in JWT_SECRET in $APP_ENV_FILE." >&2
  echo "Generate one with: openssl rand -hex 32" >&2
  exit 1
fi

if [[ "$PUBLIC_BASE_URL" == http://localhost* || "$PUBLIC_BASE_URL" == http://127.0.0.1* ]]; then
  echo "Warning: PUBLIC_BASE_URL is $PUBLIC_BASE_URL." >&2
  echo "Verification emails will link to that address." >&2
fi

for tool in aws sam go; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing required tool: $tool" >&2
    exit 1
  fi
done

if ! identity="$(aws sts get-caller-identity --query 'Arn' --output text 2>/dev/null)"; then
  echo "AWS authentication failed. Check $AWS_ENV_FILE and run:" >&2
  echo "  aws sso login --profile \"${AWS_PROFILE:-default}\"" >&2
  echo "or configure the profile first:" >&2
  echo "  aws configure sso --profile \"${AWS_PROFILE:-default}\"" >&2
  exit 1
fi
echo "Deploying as: $identity"

if [[ -n "${S3_BUCKET:-}" ]]; then
  if ! aws s3api head-bucket --bucket "$S3_BUCKET" --region "$AWS_REGION" >/dev/null 2>&1; then
    if ((CREATE_BUCKET)); then
      echo "Creating S3 bucket $S3_BUCKET in $AWS_REGION..."
      create_args=(s3api create-bucket --bucket "$S3_BUCKET" --region "$AWS_REGION")
      if [[ "$AWS_REGION" != "us-east-1" ]]; then
        create_args+=(--create-bucket-configuration "LocationConstraint=$AWS_REGION")
      fi

      if ! create_output="$(aws "${create_args[@]}" 2>&1)"; then
        if aws s3api head-bucket --bucket "$S3_BUCKET" --region "$AWS_REGION" >/dev/null 2>&1; then
          echo "S3 bucket $S3_BUCKET is now available."
        else
          echo "Could not create or access S3 bucket $S3_BUCKET in $AWS_REGION." >&2
          echo "$create_output" >&2
          echo "Create it with an admin identity, or grant s3:CreateBucket and retry." >&2
          exit 1
        fi
      else
        echo "Created S3 bucket $S3_BUCKET."
      fi
    else
      echo "S3 bucket $S3_BUCKET is missing or not accessible in $AWS_REGION." >&2
      echo "Create it once with:" >&2
      if [[ "$AWS_REGION" == "us-east-1" ]]; then
        echo "  aws s3api create-bucket --bucket \"$S3_BUCKET\" --region us-east-1" >&2
      else
        echo "  aws s3api create-bucket --bucket \"$S3_BUCKET\" --region \"$AWS_REGION\" --create-bucket-configuration LocationConstraint=$AWS_REGION" >&2
      fi
      echo "Or rerun without --no-create-bucket." >&2
      exit 1
    fi
  fi
fi

echo "Building Lambda..."
sam build

deploy_args=(
  --stack-name "$STACK_NAME"
  --region "$AWS_REGION"
  --parameter-overrides
  "MongoDBURI=$MONGODB_URI"
  "JwtSecret=$JWT_SECRET"
  "PublicBaseURL=$PUBLIC_BASE_URL"
  "EmailFrom=$EMAIL_FROM"
  "SmtpAddr=$SMTP_ADDR"
)

if [[ -n "${S3_BUCKET:-}" ]]; then
  deploy_args+=(--s3-bucket "$S3_BUCKET" --no-resolve-s3)
fi

if [[ -n "${SMTP_USERNAME:-}" ]]; then
  deploy_args+=("SmtpUsername=$SMTP_USERNAME" "SmtpPassword=$SMTP_PASSWORD")
fi

if [[ -n "${CORS_ALLOWED_ORIGINS:-}" ]]; then
  deploy_args+=("CorsAllowedOrigins=$CORS_ALLOWED_ORIGINS")
fi

if ((ASSUME_YES)); then
  deploy_args+=(--no-confirm-changeset)
fi
deploy_args+=(--no-fail-on-empty-changeset)

echo "Deploying stack $STACK_NAME..."
sam deploy "${deploy_args[@]}"

if ((SKIP_INDEXES)); then
  echo "Skipping MongoDB indexes."
else
  echo "Ensuring MongoDB indexes..."
  go run ./cmd/indexes
fi

API_URL="$(
  aws cloudformation describe-stacks \
    --region "$AWS_REGION" \
    --stack-name "$STACK_NAME" \
    --query "Stacks[0].Outputs[?OutputKey=='FunctionUrl'].OutputValue" \
    --output text
)"

if [[ -z "$API_URL" || "$API_URL" == "None" ]]; then
  echo "Could not read FunctionUrl from stack $STACK_NAME." >&2
  exit 1
fi
[[ "$API_URL" == */ ]] || API_URL="$API_URL/"

echo "Checking ${API_URL}ping ..."
if ! curl -fsS "${API_URL}ping" >/dev/null; then
  echo "The function URL is up, but /ping failed." >&2
  exit 1
fi

ready=0
for _ in 1 2 3 4 5; do
  if curl -fsS "${API_URL}ready" >/dev/null; then
    ready=1
    break
  fi
  sleep 2
done

if ((ready == 0)); then
  echo "The function URL is up, but /ready did not pass." >&2
  echo "Check the MongoDB Atlas allowlist and the Lambda logs:" >&2
  echo "  sam logs -n ApiFunction --stack-name $STACK_NAME --region $AWS_REGION --tail" >&2
  exit 1
fi

echo
echo "Deployed: $API_URL"
echo "Logs: sam logs -n ApiFunction --stack-name $STACK_NAME --region $AWS_REGION --tail"

if ((WATCH)); then
  echo
  echo "Starting sam sync --watch. Press Ctrl+C to stop."
  exec sam sync --stack-name "$STACK_NAME" --region "$AWS_REGION" --watch
fi
