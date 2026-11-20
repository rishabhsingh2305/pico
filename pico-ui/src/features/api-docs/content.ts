import type { ApiParameter, ApiSection } from "@/features/api-docs/types"

const orgIdParam: ApiParameter = {
  name: "orgId",
  required: true,
  description:
    "Organization ID. Required on every org-scoped request. On /orgs/{orgID} routes it must match the path.",
}

const limitParam: ApiParameter = {
  name: "limit",
  required: false,
  description: "Page size. Default 20, maximum 100.",
}

const offsetParam: ApiParameter = {
  name: "offset",
  required: false,
  description: "Number of matching records to skip. Default 0.",
}

const userExample = `{
  "id": "64f1a2b3c4d5e6f7890a1234",
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "emailVerified": true,
  "createdAt": "2026-01-02T12:30:00Z",
  "updatedAt": "2026-01-02T12:30:00Z"
}`

const orgExample = `{
  "id": "64f1a2b3c4d5e6f7890a1234",
  "name": "Acme Inc.",
  "userId": "64f1a2b3c4d5e6f7890a5678",
  "tags": ["team", "finance"],
  "createdAt": "2026-01-02T12:30:00Z",
  "updatedAt": "2026-01-02T12:30:00Z"
}`

const projectExample = `{
  "id": "64f1a2b3c4d5e6f7890a1234",
  "orgId": "64f1a2b3c4d5e6f7890a5678",
  "userId": "64f1a2b3c4d5e6f7890a9012",
  "name": "Launch",
  "currency": "USD",
  "tags": ["travel", "food"],
  "createdAt": "2026-01-02T12:30:00Z",
  "updatedAt": "2026-01-02T12:30:00Z"
}`

const expenseExample = `{
  "id": "64f1a2b3c4d5e6f7890a1234",
  "orgId": "64f1a2b3c4d5e6f7890a5678",
  "userId": "64f1a2b3c4d5e6f7890a9012",
  "projectId": "64f1a2b3c4d5e6f7890a3456",
  "type": "debit",
  "amount": 1299,
  "currency": "USD",
  "description": "Lunch",
  "tags": ["food"],
  "spentAt": "2026-01-02T12:30:00Z",
  "createdAt": "2026-01-02T12:30:00Z",
  "updatedAt": "2026-01-02T12:30:00Z"
}`

const memberExample = `{
  "id": "64f1a2b3c4d5e6f7890a1234",
  "orgId": "64f1a2b3c4d5e6f7890a5678",
  "userId": "64f1a2b3c4d5e6f7890a9012",
  "owner": false,
  "roles": {
    "member": "viewer",
    "project": "editor",
    "expense": "editor"
  },
  "createdAt": "2026-01-02T12:30:00Z"
}`

const linkExample = `{
  "id": "64f1a2b3c4d5e6f7890a1234",
  "orgId": "64f1a2b3c4d5e6f7890a5678",
  "userId": "64f1a2b3c4d5e6f7890a9012",
  "slug": "launch",
  "url": "https://example.com/page",
  "title": "Launch page",
  "tags": ["marketing"],
  "shortPath": "/s/launch",
  "shortUrl": "https://link.uppixl.com/s/launch",
  "disabled": false,
  "createdAt": "2026-11-20T10:00:00Z",
  "updatedAt": "2026-11-20T10:00:00Z"
}`

const errorsNote =
  'Errors use the shape {"error":"message"}. Internal details are never returned.'

export const API_SECTIONS: ApiSection[] = [
  {
    id: "health",
    title: "Health",
    description:
      "Liveness and database readiness checks. These endpoints do not require identity.",
    endpoints: [
      {
        id: "health-ping-get",
        method: "GET",
        path: "/ping",
        title: "Check liveness",
        summary:
          "Returns a small response when the HTTP process is reachable. It does not query MongoDB.",
        access: "Public",
        response: {
          status: 200,
          description: "Service is alive.",
          body: '{\n  "message": "pong"\n}',
        },
      },
      {
        id: "health-ping-head",
        method: "HEAD",
        path: "/ping",
        title: "Check liveness headers",
        summary: "Same as GET /ping, with an empty response body.",
        access: "Public",
        response: {
          status: 200,
          description: "Empty body.",
        },
      },
      {
        id: "health-ready-get",
        method: "GET",
        path: "/ready",
        title: "Check database readiness",
        summary:
          "Pings MongoDB with a 3 second timeout. Use this before sending traffic, not as a high-frequency liveness check.",
        access: "Public",
        response: {
          status: 200,
          description: "MongoDB is reachable.",
          body: '{\n  "status": "ok"\n}',
        },
        errors: ["503 — database unavailable"],
        notes: ["The underlying database error is logged, not returned."],
      },
      {
        id: "health-ready-head",
        method: "HEAD",
        path: "/ready",
        title: "Check database readiness headers",
        summary: "Same checks as GET /ready, with an empty response body.",
        access: "Public",
        response: {
          status: 200,
          description: "Empty body.",
        },
        errors: ["503 — database unavailable"],
      },
    ],
  },
  {
    id: "authentication",
    title: "Authentication",
    description:
      "Login returns a 30-minute JWT access token, a 30-day refresh token, and HttpOnly cookie fallbacks. Send the access token as Authorization: Bearer. On 401, exchange the refresh token at /auth/refresh.",
    endpoints: [
      {
        id: "auth-login",
        method: "POST",
        path: "/auth/login",
        title: "Sign in",
        summary:
          "Verifies an email and password and returns an access token. The email must already be verified.",
        access: "Public",
        requestBody: `{
  "email": "ada@example.com",
  "password": "correct horse battery staple"
}`,
        response: {
          status: 200,
          description:
            "Signed in. Send accessToken as Authorization: Bearer. pico_auth and pico_refresh cookies are also set as fallbacks.",
          body: `{
  "accessToken": "eyJ...",
  "refreshToken": "eyJ...",
  "tokenType": "Bearer",
  "user": ${userExample.replace(/\n/g, "\n  ")},
  "expiresAt": "2026-01-02T13:00:00Z",
  "refreshExpiresAt": "2026-02-01T12:30:00Z",
  "orgs": {
    "org-1": ["*"],
    "org-2": ["org.viewer", "project.viewer", "expense.viewer"]
  }
}`,
        },
        errors: [
          "400 — validation error or malformed JSON",
          "401 — invalid email or password",
          "403 — the email address is not verified",
        ],
        notes: [
          "Unknown emails run a dummy bcrypt comparison so response timing does not reveal whether an account exists.",
          "Access tokens expire after 30 minutes and carry the user's feature roles for each organization. Refresh tokens expire after 30 days.",
          "Tokens are stateless and cannot be revoked before they expire. Signing out clears the stored tokens and cookies.",
        ],
      },
      {
        id: "auth-refresh",
        method: "POST",
        path: "/auth/refresh",
        title: "Refresh tokens",
        summary:
          "Exchanges a refresh token for a new access token and a rotated refresh token. Memberships are re-queried, so role changes take effect.",
        access: "Public",
        requestBody: `{
  "refreshToken": "eyJ..."
}`,
        response: {
          status: 200,
          description:
            "New token pair. The refresh token may also be supplied through the pico_refresh cookie.",
          body: `{
  "accessToken": "eyJ...",
  "refreshToken": "eyJ...",
  "tokenType": "Bearer",
  "user": ${userExample.replace(/\n/g, "\n  ")},
  "expiresAt": "2026-01-02T13:30:00Z",
  "refreshExpiresAt": "2026-02-01T13:00:00Z"
}`,
        },
        errors: ["401 — the refresh token is missing, invalid, or expired"],
        notes: [
          "Refresh tokens are stateless and rotation does not invalidate the previous refresh token.",
        ],
      },
      {
        id: "auth-logout",
        method: "POST",
        path: "/auth/logout",
        title: "Sign out",
        summary: "Clears the pico_auth and pico_refresh cookie fallbacks.",
        access: "Public",
        response: {
          status: 204,
          description: "Signed out. No response body.",
        },
      },
      {
        id: "auth-session",
        method: "GET",
        path: "/auth/session",
        title: "Get the current session",
        summary:
          "Validates Authorization: Bearer (or the cookie fallback) and returns the current user and token expiry.",
        access: "User",
        response: {
          status: 200,
          description: "The current user.",
          body: `{
  "accessToken": "eyJ...",
  "tokenType": "Bearer",
  "user": ${userExample.replace(/\n/g, "\n  ")},
  "expiresAt": "2026-01-02T13:00:00Z",
  "orgs": {
    "org-1": ["*"],
    "org-2": ["org.viewer", "project.viewer", "expense.viewer"]
  }
}`,
        },
        errors: [
          "401 — the bearer token and cookie are missing, invalid, or expired",
        ],
      },
    ],
  },
  {
    id: "users",
    title: "Users",
    description:
      "Registration, profile reads, profile changes, and password changes. Passwords are bcrypt-hashed and never returned.",
    endpoints: [
      {
        id: "users-create",
        method: "POST",
        path: "/users",
        title: "Register a user",
        summary:
          "Creates an unverified user and sends an email verification link.",
        access: "Public",
        requestBody: `{
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "password": "correct horse battery staple"
}`,
        response: {
          status: 201,
          description: `Created. The user starts with emailVerified: false.`,
          body: userExample.replace(
            '"emailVerified": true',
            '"emailVerified": false'
          ),
        },
        errors: [
          "400 — validation error or malformed JSON",
          "409 — an account with this email already exists",
          "502 — the verification email could not be sent",
        ],
        notes: [
          "Names are capped at 120 bytes, emails at 254 bytes, and passwords at 8–72 bytes.",
          "If email delivery fails, the newly created user is removed and 502 is returned.",
        ],
      },
      {
        id: "users-update",
        method: "PATCH",
        path: "/users/me",
        title: "Update the current user",
        summary: "Updates the authenticated user's name, email, or both.",
        access: "User",
        requestBody: `{
  "name": "Ada Lovelace",
  "email": "ada@example.com"
}`,
        response: {
          status: 200,
          description: "The updated user.",
          body: userExample,
        },
        errors: [
          "400 — validation error or malformed JSON",
          "404 — user not found",
          "409 — the new email is already in use",
        ],
        notes: [
          "Changing the email marks the account unverified and sends a new verification email.",
        ],
      },
      {
        id: "users-password",
        method: "POST",
        path: "/users/me/password",
        title: "Change password",
        summary:
          "Changes the authenticated user's password after verifying the current password.",
        access: "User",
        requestBody: `{
  "currentPassword": "correct horse battery staple",
  "newPassword": "a new correct horse battery staple"
}`,
        response: {
          status: 204,
          description: "Password changed. No response body.",
        },
        errors: [
          "400 — current password is incorrect or the new password is invalid",
          "404 — user not found",
        ],
      },
    ],
  },
  {
    id: "email-verification",
    title: "Email verification",
    description:
      "Verification emails contain a signed JWT link valid for 24 hours and a 6-digit code valid for 10 minutes. Email verification proves email ownership, but it is not an access token.",
    endpoints: [
      {
        id: "auth-verify-email",
        method: "POST",
        path: "/auth/verify-email",
        title: "Verify an email",
        summary:
          "Validates the token from the verification link and marks the matching email as verified.",
        access: "Public",
        requestBody: `{\n  "token": "verification-token"\n}`,
        response: {
          status: 200,
          description: "The email is verified. Verification is idempotent.",
          body: `{\n  "status": "verified",\n  "userId": "64f1a2b3c4d5e6f7890a5678",\n  "email": "ada@example.com"\n}`,
        },
        errors: [
          "400 — missing, invalid, or expired token",
          "500 — unexpected verification failure",
        ],
        notes: [
          "The token is sent in the JSON body, not as a query parameter.",
          "This response is not an API credential and must not be used for authorization.",
        ],
      },
      {
        id: "auth-verify-email-otp",
        method: "POST",
        path: "/auth/verify-email/otp",
        title: "Verify an email with a code",
        summary:
          "Validates the 6-digit code from the verification email and marks the matching email as verified.",
        access: "Public",
        requestBody: `{\n  "email": "ada@example.com",\n  "code": "123456"\n}`,
        response: {
          status: 200,
          description: "The email is verified. Verification is idempotent.",
          body: `{\n  "status": "verified",\n  "userId": "64f1a2b3c4d5e6f7890a5678",\n  "email": "ada@example.com"\n}`,
        },
        errors: [
          "400 — invalid, expired, or missing code",
          "400 — too many failed attempts",
          "500 — unexpected verification failure",
        ],
        notes: [
          "Codes expire after 10 minutes and allow at most 5 failed attempts.",
          "Resending a verification email rotates the code.",
        ],
      },
      {
        id: "auth-resend-verification",
        method: "POST",
        path: "/auth/verify-email/resend",
        title: "Resend verification email",
        summary:
          "Sends another verification email with a new link and code to a registered, unverified address.",
        access: "Public",
        requestBody: `{\n  "email": "ada@example.com"\n}`,
        response: {
          status: 202,
          description:
            "Request accepted. The response is intentionally generic for unknown and already-verified addresses. Retry-After: 60 is also returned.",
          body: '{\n  "status": "sent",\n  "retryAfterSeconds": 60\n}',
        },
        errors: ["400 — malformed email or invalid JSON"],
        notes: [
          "Resend is throttled to once per 60 seconds per user.",
          "Unknown and already-verified addresses still return 202 to prevent account enumeration.",
          "Each resend rotates the 6-digit code.",
          "Retry-After and retryAfterSeconds are always 60, even when no email was sent, to avoid enumeration.",
        ],
      },
    ],
  },
  {
    id: "organizations",
    title: "Organizations",
    description:
      "Organizations are tenants. Creating one also creates the owner's membership. The owner is stored on the organization and the owner membership row is created at the same time.",
    endpoints: [
      {
        id: "orgs-create",
        method: "POST",
        path: "/orgs",
        title: "Create an organization",
        summary:
          "Creates an organization owned by the actor and adds the owner membership.",
        access: "User",
        requestBody: `{\n  "name": "Acme Inc.",\n  "tags": ["team", "finance"]\n}`,
        response: {
          status: 201,
          description: "Created organization. A Location header is returned.",
          body: orgExample,
        },
        errors: [
          "400 — validation error or malformed JSON",
          "404 — actor not found",
        ],
        notes: [
          "Names are required and capped at 120 bytes.",
          "Tags are optional, trimmed, deduplicated, sorted, and capped at 20 items of 32 bytes each.",
          "If the owner membership cannot be created, the organization is deleted as a best-effort cleanup.",
        ],
      },
      {
        id: "orgs-list",
        method: "GET",
        path: "/orgs?tags=...&limit=20&offset=0",
        title: "List organizations",
        summary: "Returns organizations where the actor has a membership row.",
        access: "User",
        queryParams: [
          {
            name: "tags",
            required: false,
            description:
              "Comma-separated tags. Any matching tag includes the organization.",
          },
          limitParam,
          offsetParam,
        ],
        response: {
          status: 200,
          description: "Paginated organizations.",
          body: `{\n  "items": [${orgExample}]\n}`,
        },
        notes: [
          "The owner is included through the owner membership row.",
          "Tags use any-match semantics; absent or empty tags returns all organizations.",
          "Organizations without a stored tags field read as an empty list.",
        ],
      },
      {
        id: "orgs-get",
        method: "GET",
        path: "/orgs/{orgID}?orgId=...",
        title: "Get an organization",
        summary: "Returns one organization when the actor is a member.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 200,
          description: "The requested organization.",
          body: orgExample,
        },
        errors: ["403 — not a member", "404 — organization not found"],
      },
      {
        id: "orgs-update",
        method: "PATCH",
        path: "/orgs/{orgID}?orgId=...",
        title: "Update an organization",
        summary:
          "Updates an organization's name or tags. Requires org.admin or the owner wildcard.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{\n  "name": "Acme Labs",\n  "tags": ["team", "finance"]\n}`,
        response: {
          status: 200,
          description: "Updated organization.",
          body: orgExample.replace('"Acme Inc."', '"Acme Labs"'),
        },
        errors: [
          "400 — validation error or malformed JSON",
          "403 — actor is not the owner",
          "404 — organization not found",
        ],
        notes: [
          "At least one of name or tags is required.",
          "tags: [] clears all organization tags.",
          "Only the owner may update an organization.",
        ],
      },
    ],
  },
  {
    id: "memberships",
    title: "Memberships",
    description:
      "Membership roles are feature-scoped. Each member stores a role per feature: org, member, project, and expense. The role for a feature is viewer, editor, or admin. The owner is derived from org.userId and is implicit; there are no project-level roles.",
    endpoints: [
      {
        id: "memberships-add",
        method: "POST",
        path: "/orgs/{orgID}/members?orgId=...",
        title: "Add a member",
        summary:
          "Adds an existing user to the organization. Requires member.admin or the owner wildcard.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{\n  "userId": "64f1a2b3c4d5e6f7890a9012",\n  "roles": {\n    "org": "viewer",\n    "member": "viewer",\n    "project": "editor",\n    "expense": "viewer"\n  }\n}`,
        response: {
          status: 201,
          description: "Member added.",
          body: memberExample,
        },
        errors: [
          "400 — validation error, malformed JSON, or an unknown feature or role",
          "403 — actor is not the owner",
          "404 — organization or target user not found",
          "409 — target is already a member or is the owner",
        ],
        notes: [
          "Omitted roles use the legacy editor defaults: org.viewer, member.viewer, project.editor, expense.editor.",
          "A feature may be omitted to deny access to it. Valid roles are viewer, editor, and admin, and only roles declared for that feature in features.yml are accepted.",
          "There is no invitation or acceptance flow yet; the owner adds an existing user ID.",
        ],
      },
      {
        id: "memberships-list",
        method: "GET",
        path: "/orgs/{orgID}/members?orgId=...&limit=20&offset=0",
        title: "List members",
        summary:
          "Lists memberships for an organization. The owner and roles with member.viewer or member.admin may read it.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, limitParam, offsetParam],
        response: {
          status: 200,
          description:
            "Paginated memberships. The owner has owner: true; other members have a roles map such as {project: editor, expense: viewer}.",
          body: `{\n  "items": [${memberExample}]\n}`,
        },
        errors: ["403 — not a member", "404 — organization not found"],
      },
      {
        id: "memberships-update",
        method: "PATCH",
        path: "/orgs/{orgID}/members/{memberUserID}?orgId=...",
        title: "Change member roles",
        summary:
          "Replaces a member's feature roles. Requires member.admin; the owner's roles cannot change.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
          {
            name: "memberUserID",
            required: true,
            description: "User ID of the member whose role changes.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{\n  "roles": {\n    "project": "viewer",\n    "expense": "editor"\n  }\n}`,
        response: {
          status: 204,
          description: "Roles changed. No response body.",
        },
        errors: [
          "400 — the roles map is missing or empty, or a feature or role is not declared in features.yml",
          "403 — actor is not the owner",
          "404 — organization or membership not found",
          "409 — attempting to change the owner's roles",
        ],
        notes: [
          "The request replaces the whole roles map. Omitted features lose access.",
          "The new roles take effect after the member's access token is refreshed.",
        ],
      },
      {
        id: "memberships-remove",
        method: "DELETE",
        path: "/orgs/{orgID}/members/{memberUserID}?orgId=...",
        title: "Remove a member",
        summary:
          "Removes a member. Requires member.admin; the owner cannot be removed.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
          {
            name: "memberUserID",
            required: true,
            description: "User ID of the member to remove.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 204,
          description: "Membership removed. No response body.",
        },
        errors: [
          "403 — actor is not the owner",
          "404 — organization or membership not found",
          "409 — attempting to remove the owner",
        ],
      },
    ],
  },
  {
    id: "sharing",
    title: "Sharing",
    description:
      "Share links are opaque, read-only credentials scoped to an organization, project, or expense. The raw token is returned once. Recipients send Authorization: Share <token> on the existing read routes.",
    endpoints: [
      {
        id: "shares-create",
        method: "POST",
        path: "/orgs/{orgID}/shares?orgId=...",
        title: "Create a share link",
        summary:
          "Creates a read-only link for an organization, project, or expense. Requires shares.create.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{
  "resourceType": "project",
  "resourceId": "project-1",
  "name": "Client review",
  "expiresAt": "2026-12-31T23:59:59Z"
}`,
        response: {
          status: 201,
          description:
            "Share created. token and sharePath are returned only once.",
          body: `{
  "id": "cred-1",
  "kind": "share",
  "orgId": "org-1",
  "resourceType": "project",
  "resourceId": "project-1",
  "projectId": "project-1",
  "scopes": ["orgs.get", "projects.get", "expenses.list"],
  "createdAt": "2026-11-16T10:00:00Z",
  "expiresAt": "2026-12-31T23:59:59Z",
  "status": "active",
  "token": "pico_share_...",
  "sharePath": "/shared?orgId=org-1&projectId=project-1&resourceId=project-1&resourceType=project#share=pico_share_..."
}`,
        },
        errors: [
          "400 — invalid resourceType, resourceId, expiry, or name",
          "403 — missing shares.create",
          "404 — organization, project, or expense not found",
        ],
        notes: [
          "resourceType is org, project, or expense.",
          "Scopes are derived server-side.",
          "expiresAt defaults to 30 days and is capped at 365 days.",
        ],
      },
      {
        id: "shares-list",
        method: "GET",
        path: "/orgs/{orgID}/shares?orgId=...&limit=20&offset=0",
        title: "List share links",
        summary: "Lists share metadata. Raw tokens are never returned.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, limitParam, offsetParam],
        response: {
          status: 200,
          description: "Share metadata without tokens.",
          body: `{\n  "items": []\n}`,
        },
        errors: ["403 — missing shares.list"],
      },
      {
        id: "shares-status",
        method: "GET",
        path: "/orgs/{orgID}/shares/status?orgId=...&resourceType=expense&resourceIds=...",
        title: "Check share status",
        summary: "Returns active share counts for a page of resource IDs.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [
          orgIdParam,
          {
            name: "resourceType",
            required: true,
            description: "org, project, or expense.",
          },
          {
            name: "resourceIds",
            required: true,
            description: "Comma-separated IDs, at most 100.",
          },
        ],
        response: {
          status: 200,
          description:
            "Active shares per requested resource. Missing IDs are not shared.",
          body: `{\n  "resourceType": "expense",\n  "items": [\n    { "resourceId": "expense-42", "activeCount": 1, "expiresAt": "2026-12-31T23:59:59Z" }\n  ]\n}`,
        },
        errors: [
          "400 — missing or invalid filters",
          "403 — missing shares.status",
        ],
      },
      {
        id: "shares-credentials",
        method: "GET",
        path: "/orgs/{orgID}/credentials?orgId=...&kind=share&resourceType=...&resourceIds=...&active=true",
        title: "Filter credentials",
        summary:
          "Filters shares and future API keys by kind, resource type, resource IDs, and active state.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [
          orgIdParam,
          {
            name: "kind",
            required: false,
            description: "share (default) or api_key.",
          },
          {
            name: "resourceType",
            required: false,
            description: "org, project, or expense; share credentials only.",
          },
          {
            name: "resourceIds",
            required: false,
            description: "Comma-separated IDs, at most 100.",
          },
          {
            name: "active",
            required: false,
            description: "true returns only active credentials.",
          },
        ],
        response: {
          status: 200,
          description: "Matching credential metadata without tokens.",
          body: `{\n  "items": []\n}`,
        },
        errors: [
          "400 — invalid kind or filters",
          "403 — missing shares.credentials",
        ],
      },
      {
        id: "shares-revoke",
        method: "DELETE",
        path: "/orgs/{orgID}/shares/{shareID}?orgId=...",
        title: "Revoke a share link",
        summary: "Revokes a link immediately. Requires shares.revoke.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
          {
            name: "shareID",
            required: true,
            description: "Credential MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 204,
          description: "Revoked. No response body.",
        },
        errors: ["403 — missing shares.revoke", "404 — share not found"],
      },
    ],
  },
  {
    id: "links",
    title: "Links",
    description:
      "Short links redirect publicly at /s/{slug}. Management is organization-scoped; creating, editing, disabling, and deleting require link.* roles.",
    endpoints: [
      {
        id: "links-create",
        method: "POST",
        path: "/orgs/{orgID}/links?orgId=...",
        title: "Create a short link",
        summary:
          "Creates a public redirect. An omitted slug generates a random 7-character alias. Requires links.create.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{
  "url": "https://example.com/page",
  "slug": "launch",
  "title": "Launch page",
  "tags": ["marketing"]
}`,
        response: {
          status: 201,
          description:
            "Link created. shortUrl uses SHORT_BASE_URL when configured, otherwise the API request origin.",
          body: linkExample,
        },
        errors: [
          "400 — invalid URL, alias, title, or tags",
          "403 — missing links.create",
          "409 — alias already in use",
        ],
        notes: [
          "Only http and https destinations are accepted.",
          "Aliases are lowercase, 3–32 characters, and cannot be changed later.",
          "The alias is globally unique because /s/{slug} has no organization context.",
        ],
      },
      {
        id: "links-list",
        method: "GET",
        path: "/orgs/{orgID}/links?orgId=...&q=...&tags=...&limit=20&offset=0",
        title: "List short links",
        summary:
          "Lists one page of links. q is an alias prefix; tags use any-match semantics. Requires links.list.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [
          orgIdParam,
          {
            name: "q",
            required: false,
            description:
              "Alias prefix, case-insensitive, maximum 64 characters.",
          },
          {
            name: "tags",
            required: false,
            description: "Comma-separated tags; any tag may match.",
          },
          limitParam,
          offsetParam,
        ],
        response: {
          status: 200,
          description:
            "One page of links. nextOffset is null on the last page.",
          body: `{\n  "items": [],\n  "nextOffset": 20\n}`,
        },
        errors: [
          "400 — invalid pagination or query",
          "403 — missing links.list",
        ],
      },
      {
        id: "links-get",
        method: "GET",
        path: "/orgs/{orgID}/links/{linkID}?orgId=...",
        title: "Get a short link",
        summary:
          "Returns one link, including disabled links. Requires links.get.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
          {
            name: "linkID",
            required: true,
            description: "Link MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 200,
          description: "The link record.",
          body: linkExample,
        },
        errors: ["403 — missing links.get", "404 — link not found"],
      },
      {
        id: "links-update",
        method: "PATCH",
        path: "/orgs/{orgID}/links/{linkID}?orgId=...",
        title: "Update a short link",
        summary:
          "Updates the destination, title, tags, or disabled state. The slug cannot change. Requires links.update.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
          {
            name: "linkID",
            required: true,
            description: "Link MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{
  "url": "https://example.com/new-page",
  "title": "New title",
  "tags": ["marketing", "launch"],
  "disabled": true
}`,
        response: {
          status: 200,
          description: "The updated link record.",
          body: linkExample,
        },
        errors: [
          "400 — invalid or missing fields",
          "403 — missing links.update",
          "404 — link not found",
        ],
        notes: [
          "At least one field is required.",
          "disabled: true stops the redirect; disabled: false re-enables it.",
          "Sending slug is rejected.",
        ],
      },
      {
        id: "links-delete",
        method: "DELETE",
        path: "/orgs/{orgID}/links/{linkID}?orgId=...",
        title: "Delete a short link",
        summary:
          "Tombstones the link. It disappears from lists and the alias stays reserved. Requires links.delete.",
        access: "Org role",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
          {
            name: "linkID",
            required: true,
            description: "Link MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 204,
          description: "Link deleted; empty body.",
        },
        errors: ["403 — missing links.delete", "404 — link not found"],
      },
      {
        id: "links-redirect",
        method: "GET",
        path: "/s/{slug}",
        title: "Follow a short link",
        summary:
          "Public redirect. No authentication, organization membership, or token is required.",
        access: "Public",
        pathParams: [
          {
            name: "slug",
            required: true,
            description: "Link alias.",
          },
        ],
        response: {
          status: 302,
          description:
            "Redirects to the destination. Cache-Control: public, max-age=0, s-maxage=600 and X-Robots-Tag: noindex are set.",
        },
        errors: ["404 — link is missing, disabled, or deleted"],
        notes: [
          "HEAD follows the same behavior with an empty body.",
          "The destination can be changed or the link disabled at any time.",
        ],
      },
    ],
  },
  {
    id: "projects",
    title: "Projects",
    description:
      "Projects belong to an organization and own the currency used by expenses. Project tags organize projects and use any-match filtering.",
    endpoints: [
      {
        id: "projects-create",
        method: "POST",
        path: "/projects?orgId=...",
        title: "Create a project",
        summary:
          "Creates a project with a name, immutable currency, and optional tags.",
        access: "Org role",
        queryParams: [orgIdParam],
        requestBody: `{
  "name": "Launch",
  "currency": "USD",
  "tags": ["travel", "food"]
}`,
        response: {
          status: 201,
          description: "Created project. A Location header is returned.",
          body: projectExample,
        },
        errors: [
          "400 — validation error or malformed JSON",
          "403 — not a member",
          "404 — organization not found",
        ],
        notes: [
          "Currency is a three-letter ISO code and cannot change after creation.",
          "Tags are trimmed, deduplicated, sorted, and capped at 20 items of 32 bytes each.",
        ],
      },
      {
        id: "projects-list",
        method: "GET",
        path: "/projects?orgId=...&tags=...&limit=20&offset=0",
        title: "List projects",
        summary:
          "Returns projects in the organization. Tag filtering uses any-match semantics.",
        access: "Org role",
        queryParams: [
          orgIdParam,
          {
            name: "tags",
            required: false,
            description:
              "Comma-separated tags. A project matches when it has any requested tag.",
          },
          limitParam,
          offsetParam,
        ],
        response: {
          status: 200,
          description: "Paginated projects.",
          body: `{\n  "items": [${projectExample}]\n}`,
        },
        errors: ["403 — not a member", "404 — organization not found"],
        notes: [
          "tags=food,travel returns projects with food, travel, or both.",
          "This is intentionally different from expense tag filtering, which requires every tag.",
        ],
      },
      {
        id: "projects-get",
        method: "GET",
        path: "/projects/{projectID}?orgId=...",
        title: "Get a project",
        summary: "Returns one project scoped to the organization.",
        access: "Org role",
        pathParams: [
          {
            name: "projectID",
            required: true,
            description: "Project MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 200,
          description: "The requested project.",
          body: projectExample,
        },
        errors: ["403 — not a member", "404 — project not found"],
      },
      {
        id: "projects-update",
        method: "PATCH",
        path: "/projects/{projectID}?orgId=...",
        title: "Update a project",
        summary: "Updates the project name, tags, or both.",
        access: "Org role",
        pathParams: [
          {
            name: "projectID",
            required: true,
            description: "Project MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{
  "name": "Renamed project",
  "tags": ["new", "tags"]
}`,
        response: {
          status: 200,
          description: "Updated project.",
          body: projectExample,
        },
        errors: [
          "400 — no fields were supplied or validation failed",
          "403 — not a member",
          "404 — project not found",
        ],
        notes: ["Send tags: [] to clear all project tags."],
      },
    ],
  },
  {
    id: "expenses",
    title: "Expenses",
    description:
      "Expenses belong to a project and inherit its currency. Each entry is a debit (money out) or credit (money in); amounts are positive integer minor units. Expense tag filtering requires every listed tag to match.",
    endpoints: [
      {
        id: "expenses-create",
        method: "POST",
        path: "/expenses?orgId=...",
        title: "Create an expense",
        summary:
          "Records a debit or credit in a project. The project must belong to the organization.",
        access: "Org role",
        queryParams: [orgIdParam],
        requestBody: `{
  "projectId": "64f1a2b3c4d5e6f7890a3456",
  "type": "debit",
  "amount": 1299,
  "description": "Lunch",
  "tags": ["food"],
  "spentAt": "2026-01-02T12:30:00Z"
}`,
        response: {
          status: 201,
          description: "Created expense. Currency is copied from the project.",
          body: expenseExample,
        },
        errors: [
          "400 — validation error or malformed JSON",
          "403 — not a member",
          "404 — project not found in this organization",
        ],
        notes: [
          "type is debit for money out or credit for money in. It defaults to debit when omitted.",
          "amount is a positive integer in minor units: 12.99 USD is 1299. type carries the direction.",
          "spentAt is required and uses RFC 3339.",
          "Tags are trimmed, deduplicated, sorted, and capped at 20 items.",
        ],
      },
      {
        id: "expenses-list",
        method: "GET",
        path: "/expenses?orgId=...&projectId=...&tags=...&from=...&to=...&limit=20&offset=0",
        title: "List expenses",
        summary: "Returns one page of expenses.",
        access: "Org role",
        queryParams: [
          orgIdParam,
          {
            name: "projectId",
            required: false,
            description: "Limit results to one project.",
          },
          {
            name: "projectIds",
            required: false,
            description:
              "Comma-separated project IDs, up to 100. Do not combine with projectId.",
          },
          {
            name: "tags",
            required: false,
            description:
              "Comma-separated tags. Every listed tag must be present.",
          },
          {
            name: "tag",
            required: false,
            description: "Legacy single-tag filter.",
          },
          {
            name: "from",
            required: false,
            description: "Inclusive RFC 3339 lower bound for spentAt.",
          },
          {
            name: "to",
            required: false,
            description: "Inclusive RFC 3339 upper bound for spentAt.",
          },
          limitParam,
          offsetParam,
        ],
        response: {
          status: 200,
          description:
            "The current page and the offset for the next page, or null on the last page.",
          body: `{
  "items": [${expenseExample}],
  "nextOffset": 20
}`,
        },
        errors: [
          "400 — invalid date or pagination",
          "403 — not a member",
          "404 — organization not found",
        ],
        notes: [
          "tags=food,travel requires both tags.",
          "nextOffset is null when there are no more results.",
          "Use GET /expenses/summary for totals over the full filtered set.",
          "Records without a type are treated as debit.",
        ],
      },
      {
        id: "expenses-summary",
        method: "GET",
        path: "/expenses/summary?orgId=...&projectId=...&tags=...&from=...&to=...",
        title: "Summarize expenses",
        summary:
          "Returns expense totals grouped by project and currency for the full filtered set.",
        access: "Org role",
        queryParams: [
          orgIdParam,
          {
            name: "projectId",
            required: false,
            description: "Limit the summary to one project.",
          },
          {
            name: "tags",
            required: false,
            description:
              "Comma-separated tags. Every listed tag must be present.",
          },
          {
            name: "from",
            required: false,
            description: "Inclusive RFC 3339 lower bound for spentAt.",
          },
          {
            name: "to",
            required: false,
            description: "Inclusive RFC 3339 upper bound for spentAt.",
          },
        ],
        response: {
          status: 200,
          description: "One totals row per project and currency.",
          body: `{
  "items": [
    {
      "projectId": "64f...",
      "currency": "USD",
      "totalCount": 42,
      "totalDebit": 100000,
      "totalCredit": 123456,
      "netTotal": 23456
    }
  ]
}`,
        },
        errors: [
          "400 — invalid date",
          "403 — not a member",
          "404 — organization not found",
        ],
        notes: [
          "Totals cover every matching expense; this endpoint has no pagination.",
          "Amounts are grouped by project and currency and never added across currencies.",
          "netTotal is totalCredit - totalDebit.",
          "Records without a type are treated as debit.",
        ],
      },
      {
        id: "expenses-get",
        method: "GET",
        path: "/expenses/{expenseID}?orgId=...",
        title: "Get an expense",
        summary: "Returns one expense scoped to the organization.",
        access: "Org role",
        pathParams: [
          {
            name: "expenseID",
            required: true,
            description: "Expense MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 200,
          description: "The requested expense.",
          body: expenseExample,
        },
        errors: ["403 — not a member", "404 — expense not found"],
      },
      {
        id: "expenses-update",
        method: "PATCH",
        path: "/expenses/{expenseID}?orgId=...",
        title: "Update an expense",
        summary: "Applies partial changes to an expense.",
        access: "Org role",
        pathParams: [
          {
            name: "expenseID",
            required: true,
            description: "Expense MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        requestBody: `{
  "type": "credit",
  "amount": 1599,
  "description": "Lunch with the team",
  "tags": ["food", "team"],
  "spentAt": "2026-01-03T13:00:00Z"
}`,
        response: {
          status: 200,
          description: "Updated expense.",
          body: expenseExample,
        },
        errors: [
          "400 — no fields were supplied or validation failed",
          "403 — not a member",
          "404 — expense or project not found",
        ],
        notes: [
          "Currency is inherited from the project and is not accepted in the request body.",
          "type is optional in a partial update. When supplied it must be debit or credit.",
        ],
      },
      {
        id: "expenses-delete",
        method: "DELETE",
        path: "/expenses/{expenseID}?orgId=...",
        title: "Delete an expense",
        summary: "Permanently removes an expense from the organization.",
        access: "Org role",
        pathParams: [
          {
            name: "expenseID",
            required: true,
            description: "Expense MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam],
        response: {
          status: 204,
          description: "Expense deleted. No response body.",
        },
        errors: ["403 — not a member", "404 — expense not found"],
      },
    ],
  },
]

export const COMMON_ERRORS = [
  {
    status: "400",
    description:
      "Validation error, malformed JSON, missing identity query parameter, or invalid verification token.",
  },
  {
    status: "403",
    description:
      "The token has no role for the organization, the role lacks the route permission, or an owner-only action was attempted without owning the organization.",
  },
  {
    status: "404",
    description: "Resource missing or malformed ObjectID.",
  },
  {
    status: "405",
    description: "Method not allowed. The response includes an Allow header.",
  },
  {
    status: "409",
    description:
      "Duplicate email, duplicate membership, owner removal, or another conflict.",
  },
  {
    status: "500",
    description:
      "Unexpected server failure. Internal details are logged and never returned.",
  },
  {
    status: "502",
    description: "Verification email delivery failed.",
  },
] as const

export const API_INTRO = {
  title: "Pico API",
  summary:
    "A JSON API for users, organizations, memberships, projects, and project expenses. The browser app uses the same endpoints documented here.",
  identityNote:
    "The actor is the access token sub. Org-scoped requests require ?orgId= and a matching role in the token. There is no userId query identity.",
  errorsNote,
}
