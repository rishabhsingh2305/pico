export type Project = {
  id: string
  orgId: string
  userId: string
  name: string
  currency: string
  tags: string[]
  createdAt: string
  updatedAt: string
}

export type CreateProjectInput = {
  name: string
  currency: string
  tags: string[]
}
