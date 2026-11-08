export type SignUpInput = {
  name: string
  email: string
  password: string
}

export type LoginInput = {
  email: string
  password: string
}

export type VerifyEmailOtpInput = {
  email: string
  code: string
}

export type User = {
  id: string
  name: string
  email: string
  emailVerified: boolean
  createdAt: string
  updatedAt: string
}

export type SessionResponse = {
  accessToken: string
  tokenType: string
  user: User
  expiresAt: string
}

export type VerificationResponse = {
  status: "verified"
  userId: string
  email: string
}

export type ResendVerificationResponse = {
  status: "sent"
  retryAfterSeconds: number
}
