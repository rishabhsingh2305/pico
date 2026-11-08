import { requestJSON } from "@/lib/api-client"
import type {
  LoginInput,
  ResendVerificationResponse,
  SessionResponse,
  SignUpInput,
  User,
  VerificationResponse,
} from "@/features/auth/types"

export function signUp(input: SignUpInput): Promise<User> {
  return requestJSON<User>("/users", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
}

export function login(input: LoginInput): Promise<SessionResponse> {
  return requestJSON<SessionResponse>("/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
}

export function logout(): Promise<void> {
  return requestJSON<void>("/auth/logout", { method: "POST" })
}

export function getSession(): Promise<SessionResponse> {
  return requestJSON<SessionResponse>("/auth/session", { method: "GET" })
}

export function verifyEmailOtp(
  email: string,
  code: string
): Promise<VerificationResponse> {
  return requestJSON<VerificationResponse>("/auth/verify-email/otp", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
  })
}

export function resendVerification(
  email: string
): Promise<ResendVerificationResponse> {
  return requestJSON<ResendVerificationResponse>("/auth/verify-email/resend", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  })
}
