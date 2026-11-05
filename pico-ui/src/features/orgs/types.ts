export type Org = {
  id: string
  name: string
  userId: string
  tags: string[]
  createdAt: string
  updatedAt: string
}

export type CreateOrgInput = {
  name: string
  tags: string[]
}
