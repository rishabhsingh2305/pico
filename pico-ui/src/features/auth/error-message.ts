import { ApiError } from "@/lib/api-client"

const GENERIC_ERROR = "Something went wrong. Please try again."

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
    return "Could not reach the server. Check your connection and try again."
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
    return "Could not reach the server. Check your connection and try again."
  }

  return GENERIC_ERROR
}
