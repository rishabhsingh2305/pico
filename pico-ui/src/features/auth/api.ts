import { requestJSON } from "@/lib/api-client"
import type {
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

export function verifyEmail(token: string): Promise<VerificationResponse> {
  return requestJSON<VerificationResponse>("/auth/verify-email", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  })
}
