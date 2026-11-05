import type { SignUpInput } from "@/features/auth/types"
import { byteLength } from "@/lib/text"

const NAME_MAX_BYTES = 120
const EMAIL_MAX_BYTES = 254
const PASSWORD_MIN_BYTES = 8
const PASSWORD_MAX_BYTES = 72

const emailPattern = /^[^\s@]+@[^\s@]+$/

export type SignUpField = keyof SignUpInput
export type SignUpErrors = Partial<Record<SignUpField, string>>

export function validateSignUp(input: SignUpInput): SignUpErrors {
  const errors: SignUpErrors = {}

  const name = input.name.trim()
  if (name === "") {
    errors.name = "Enter your name."
  } else if (byteLength(name) > NAME_MAX_BYTES) {
    errors.name = "Name must be at most 120 characters."
  }

  const email = input.email.trim()
  if (email === "") {
    errors.email = "Enter your email address."
  } else if (byteLength(email) > EMAIL_MAX_BYTES) {
    errors.email = "Email must be at most 254 characters."
  } else if (!emailPattern.test(email)) {
    errors.email = "Enter a valid email address."
  }

  const passwordBytes = byteLength(input.password)
  if (passwordBytes < PASSWORD_MIN_BYTES) {
    errors.password = "Password must be at least 8 characters."
  } else if (passwordBytes > PASSWORD_MAX_BYTES) {
    errors.password = "Password must be at most 72 bytes."
  }

  return errors
}
