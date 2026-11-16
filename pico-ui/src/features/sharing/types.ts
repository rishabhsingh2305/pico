export type ShareResourceType = "org" | "project" | "expense"

export type ShareStatus = "active" | "expired" | "revoked"

export type ShareCredential = {
  id: string
  kind: "share"
  orgId: string
  resourceType: ShareResourceType
  resourceId: string
  projectId?: string
  scopes: string[]
  name?: string
  createdBy: string
  createdAt: string
  expiresAt?: string
  revokedAt?: string
  lastUsedAt?: string
  status: ShareStatus
  token?: string
  sharePath?: string
}

export type CreateShareInput = {
  resourceType: ShareResourceType
  resourceId?: string
  name?: string
  expiresAt?: string
}

export type ShareStatusItem = {
  resourceId: string
  activeCount: number
  expiresAt?: string
}
