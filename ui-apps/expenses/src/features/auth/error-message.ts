import { ApiError } from "@/lib/api-client"

const GENERIC_ERROR = "Something went wrong. Please try again."
const NETWORK_ERROR =
  "Could not reach the server. Check your connection and try again."

export function loginErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.status) {
      case 400:
        return error.message
      case 401:
        return "Email or password is incorrect."
      case 403:
        return "Verify your email before signing in."
      default:
        return GENERIC_ERROR
    }
  }

  if (error instanceof TypeError) {
    return NETWORK_ERROR
  }

  return GENERIC_ERROR
}

export function signUpErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.status) {
      case 400:
        return error.message
      case 409:
        return "An account with this email already exists."
      case 502:
        return "We couldn't send the verification email. Please try again."
      default:
        return GENERIC_ERROR
    }
  }

  if (error instanceof TypeError) {
    return NETWORK_ERROR
  }

  return GENERIC_ERROR
}

export function otpErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 400) {
      if (error.message.toLowerCase().includes("too many")) {
        return "Too many attempts. Request a new code and try again."
      }
      return "That code is invalid or has expired."
    }
    return GENERIC_ERROR
  }

  if (error instanceof TypeError) {
    return NETWORK_ERROR
  }

  return GENERIC_ERROR
}

export function resendErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 400) {
      return error.message
    }
    return GENERIC_ERROR
  }

  if (error instanceof TypeError) {
    return NETWORK_ERROR
  }

  return GENERIC_ERROR
}
