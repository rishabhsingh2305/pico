import { ApiError } from "@/lib/api-client"

const GENERIC_ERROR = "Something went wrong. Please try again."

export function expenseErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.status) {
      case 400:
        return error.message
      case 403:
        return "You do not have permission to do that."
      case 404:
        return "The expense or project is no longer available."
      default:
        return GENERIC_ERROR
    }
  }

  if (error instanceof TypeError) {
    return "Could not reach the server. Check your connection and try again."
  }

  return GENERIC_ERROR
}
