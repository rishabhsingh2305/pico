export type SignUpInput = {
  name: string
  email: string
  password: string
}

export type User = {
  id: string
  name: string
  email: string
  emailVerified: boolean
  createdAt: string
  updatedAt: string
}

export type VerificationResponse = {
  status: "verified"
  userId: string
  email: string
}
