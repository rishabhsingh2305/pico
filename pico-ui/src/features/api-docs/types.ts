export type ApiMethod = "GET" | "HEAD" | "POST" | "PATCH" | "DELETE"

export type ApiAccess = "Public" | "User" | "Member" | "Owner"

export type ApiParameter = {
  name: string
  required: boolean
  description: string
}

export type ApiResponse = {
  status: number
  description: string
  body?: string
}

export type ApiEndpoint = {
  id: string
  method: ApiMethod
  path: string
  title: string
  summary: string
  access: ApiAccess
  pathParams?: ApiParameter[]
  queryParams?: ApiParameter[]
  requestBody?: string
  response: ApiResponse
  errors?: string[]
  notes?: string[]
}

export type ApiSection = {
  id: string
  title: string
  description: string
  endpoints: ApiEndpoint[]
}
