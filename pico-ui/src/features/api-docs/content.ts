import type { ApiParameter, ApiSection } from "@/features/api-docs/types"

const userIdParam: ApiParameter = {
  name: "userId",
  required: false,
  description:
    "Temporary actor fallback. Protected routes use Authorization: Bearer (or the pico_auth cookie fallback); userId is only used when neither is present and is not secure authentication.",
}

const orgIdParam: ApiParameter = {
  name: "orgId",
  required: true,
  description:
    "Organization ID. Path-scoped org routes repeat it in the query because membership middleware reads identity from query parameters.",
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
  "role": "member",
  "createdAt": "2026-01-02T12:30:00Z"
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
      "Login returns a JWT access token and also sets an HttpOnly cookie fallback. Send the token as Authorization: Bearer. The userId query parameter remains a temporary fallback.",
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
            "Signed in. Send accessToken as Authorization: Bearer. A pico_auth cookie is also set as a fallback.",
          body: `{
  "accessToken": "eyJ...",
  "tokenType": "Bearer",
  "user": ${userExample.replace(/\n/g, "\n  ")},
  "expiresAt": "2026-01-09T12:30:00Z"
}`,
        },
        errors: [
          "400 — validation error or malformed JSON",
          "401 — invalid email or password",
          "403 — the email address is not verified",
        ],
        notes: [
          "Unknown emails run a dummy bcrypt comparison so response timing does not reveal whether an account exists.",
          "Access tokens are stateless and cannot be revoked before they expire. Signing out clears the stored token and cookie.",
        ],
      },
      {
        id: "auth-logout",
        method: "POST",
        path: "/auth/logout",
        title: "Sign out",
        summary: "Clears the pico_auth cookie fallback.",
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
  "expiresAt": "2026-01-09T12:30:00Z"
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
          description: `Created. The user starts with emailVerified: false. A Location header points to /users/{id}.`,
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
        id: "users-list",
        method: "GET",
        path: "/users?userId=...&limit=20&offset=0",
        title: "List users",
        summary:
          "Returns a paginated directory of users. This is currently a global directory.",
        access: "User",
        queryParams: [userIdParam, limitParam, offsetParam],
        response: {
          status: 200,
          description: "Paginated users.",
          body: `{\n  "items": ${userExample.replace(/\n/g, "\n  ")}\n}`,
        },
        notes: [
          "This endpoint should become scoped or admin-only once real authentication and roles exist.",
        ],
      },
      {
        id: "users-get",
        method: "GET",
        path: "/users/{userID}?userId=...",
        title: "Get a user",
        summary: "Returns one user by ID.",
        access: "User",
        pathParams: [
          {
            name: "userID",
            required: true,
            description: "User MongoDB ObjectID.",
          },
        ],
        queryParams: [userIdParam],
        response: {
          status: 200,
          description: "The requested user.",
          body: userExample,
        },
        errors: ["404 — user not found"],
      },
      {
        id: "users-update",
        method: "PATCH",
        path: "/users/{userID}?userId=...",
        title: "Update a user",
        summary:
          "Updates the authenticated actor's name, email, or both. The actor can only update their own user record.",
        access: "User",
        pathParams: [
          {
            name: "userID",
            required: true,
            description: "User MongoDB ObjectID. Must match userId.",
          },
        ],
        queryParams: [userIdParam],
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
          "403 — attempting to update another user",
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
        path: "/users/{userID}/password?userId=...",
        title: "Change password",
        summary:
          "Changes the actor's own password after verifying the current password.",
        access: "User",
        pathParams: [
          {
            name: "userID",
            required: true,
            description: "User MongoDB ObjectID. Must match userId.",
          },
        ],
        queryParams: [userIdParam],
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
          "403 — attempting to change another user's password",
          "404 — user not found",
        ],
      },
    ],
  },
  {
    id: "email-verification",
    title: "Email verification",
    description:
      "Verification links contain a signed JWT valid for 24 hours. Email verification proves email ownership, but it is not an access token.",
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
        id: "auth-resend-verification",
        method: "POST",
        path: "/auth/verify-email/resend",
        title: "Resend verification email",
        summary:
          "Sends another verification email to a registered, unverified address.",
        access: "Public",
        requestBody: `{\n  "email": "ada@example.com"\n}`,
        response: {
          status: 202,
          description:
            "Request accepted. The response is intentionally generic for unknown and already-verified addresses.",
          body: '{\n  "status": "sent"\n}',
        },
        errors: ["400 — malformed email or invalid JSON"],
        notes: [
          "Resend is throttled to once per 60 seconds per user.",
          "Unknown and already-verified addresses still return 202 to prevent account enumeration.",
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
        path: "/orgs?userId=...",
        title: "Create an organization",
        summary:
          "Creates an organization owned by the actor and adds the owner membership.",
        access: "User",
        queryParams: [userIdParam],
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
        path: "/orgs?userId=...&tags=...&limit=20&offset=0",
        title: "List organizations",
        summary: "Returns organizations where the actor has a membership row.",
        access: "User",
        queryParams: [
          userIdParam,
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
        path: "/orgs/{orgID}?orgId=...&userId=...",
        title: "Get an organization",
        summary: "Returns one organization when the actor is a member.",
        access: "Member",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
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
        path: "/orgs/{orgID}?orgId=...&userId=...",
        title: "Update an organization",
        summary:
          "Updates an organization's name or tags. Only its owner may do this.",
        access: "Owner",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
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
      "Membership is flat per organization. The owner is derived from org.userId; every other membership row is a member. There are no project-level roles.",
    endpoints: [
      {
        id: "memberships-add",
        method: "POST",
        path: "/orgs/{orgID}/members?orgId=...&userId=...",
        title: "Add a member",
        summary:
          "Adds an existing user to the organization. Only the owner may add members.",
        access: "Owner",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
        requestBody: `{\n  "userId": "64f1a2b3c4d5e6f7890a9012"\n}`,
        response: {
          status: 201,
          description: "Member added.",
          body: memberExample,
        },
        errors: [
          "400 — validation error or malformed JSON",
          "403 — actor is not the owner",
          "404 — organization or target user not found",
          "409 — target is already a member or is the owner",
        ],
        notes: [
          "There is no invitation or acceptance flow yet; the owner adds an existing user ID.",
        ],
      },
      {
        id: "memberships-list",
        method: "GET",
        path: "/orgs/{orgID}/members?orgId=...&userId=...&limit=20&offset=0",
        title: "List members",
        summary:
          "Lists memberships for an organization. Any member may read it.",
        access: "Member",
        pathParams: [
          {
            name: "orgID",
            required: true,
            description: "Organization MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam, limitParam, offsetParam],
        response: {
          status: 200,
          description: "Paginated memberships with derived roles.",
          body: `{\n  "items": [${memberExample}]\n}`,
        },
        errors: ["403 — not a member", "404 — organization not found"],
      },
      {
        id: "memberships-remove",
        method: "DELETE",
        path: "/orgs/{orgID}/members/{memberUserID}?orgId=...&userId=...",
        title: "Remove a member",
        summary:
          "Removes a member. Only the owner may do this, and the owner cannot remove themselves.",
        access: "Owner",
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
        queryParams: [orgIdParam, userIdParam],
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
    id: "projects",
    title: "Projects",
    description:
      "Projects belong to an organization and own the currency used by expenses. Project tags organize projects and use any-match filtering.",
    endpoints: [
      {
        id: "projects-create",
        method: "POST",
        path: "/projects?orgId=...&userId=...",
        title: "Create a project",
        summary:
          "Creates a project with a name, immutable currency, and optional tags.",
        access: "Member",
        queryParams: [orgIdParam, userIdParam],
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
        path: "/projects?orgId=...&userId=...&tags=...&limit=20&offset=0",
        title: "List projects",
        summary:
          "Returns projects in the organization. Tag filtering uses any-match semantics.",
        access: "Member",
        queryParams: [
          orgIdParam,
          userIdParam,
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
        path: "/projects/{projectID}?orgId=...&userId=...",
        title: "Get a project",
        summary: "Returns one project scoped to the organization.",
        access: "Member",
        pathParams: [
          {
            name: "projectID",
            required: true,
            description: "Project MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
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
        path: "/projects/{projectID}?orgId=...&userId=...",
        title: "Update a project",
        summary: "Updates the project name, tags, or both.",
        access: "Member",
        pathParams: [
          {
            name: "projectID",
            required: true,
            description: "Project MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
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
        path: "/expenses?orgId=...&userId=...",
        title: "Create an expense",
        summary:
          "Records a debit or credit in a project. The project must belong to the organization.",
        access: "Member",
        queryParams: [orgIdParam, userIdParam],
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
        path: "/expenses?orgId=...&userId=...&projectId=...&tags=...&from=...&to=...&limit=20&offset=0",
        title: "List expenses",
        summary:
          "Returns one page of expenses plus totals over every expense matching the filters.",
        access: "Member",
        queryParams: [
          orgIdParam,
          userIdParam,
          {
            name: "projectId",
            required: false,
            description: "Limit results to one project.",
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
            "The current page plus totals over the full filtered set. Totals ignore limit and offset.",
          body: `{
  "items": [${expenseExample}],
  "totalCount": 42,
  "totalDebit": 100000,
  "totalCredit": 123456,
  "netTotal": 23456
}`,
        },
        errors: [
          "400 — invalid date or pagination",
          "403 — not a member",
          "404 — organization not found",
        ],
        notes: [
          "tags=food,travel requires both tags.",
          "totalDebit and totalCredit are minor-unit sums; netTotal is totalCredit - totalDebit.",
          "Totals cover every matching expense and ignore limit and offset. When projectId is supplied, they use the project's currency.",
          "Records without a type are treated as debit.",
          "An organization-wide total across currencies is not meaningful and would need grouping by currency.",
        ],
      },
      {
        id: "expenses-get",
        method: "GET",
        path: "/expenses/{expenseID}?orgId=...&userId=...",
        title: "Get an expense",
        summary: "Returns one expense scoped to the organization.",
        access: "Member",
        pathParams: [
          {
            name: "expenseID",
            required: true,
            description: "Expense MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
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
        path: "/expenses/{expenseID}?orgId=...&userId=...",
        title: "Update an expense",
        summary: "Applies partial changes to an expense.",
        access: "Member",
        pathParams: [
          {
            name: "expenseID",
            required: true,
            description: "Expense MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
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
        path: "/expenses/{expenseID}?orgId=...&userId=...",
        title: "Delete an expense",
        summary: "Permanently removes an expense from the organization.",
        access: "Member",
        pathParams: [
          {
            name: "expenseID",
            required: true,
            description: "Expense MongoDB ObjectID.",
          },
        ],
        queryParams: [orgIdParam, userIdParam],
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
      "The actor is not a member, or attempted an owner-only action without owning the organization.",
  },
  {
    status: "404",
    description:
      "Resource missing or malformed ObjectID. Protected routes also return 404 for a missing user on the temporary userId fallback.",
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
    "The API currently accepts userId and orgId as query parameters. Membership checks are enforced, but this is not authentication: any caller can claim another user ID. Do not expose the API publicly until login and access tokens replace query identity.",
  errorsNote,
}
