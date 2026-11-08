import { ApiError } from "@/lib/api-client"

const GENERIC_ERROR = "Something went wrong. Please try again."
const NETWORK_ERROR =
  "Could not reach the server. Check your connection and try again."

export function expenseErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.status) {
      case 400:
        return error.message
      case 401:
        return "Your session expired. Please sign in again."
      case 403:
        return "You don't have access to this expense space."
      case 404:
        return "We couldn't find this category. Try reloading."
      default:
        return GENERIC_ERROR
    }
  }

  if (error instanceof TypeError) {
    return NETWORK_ERROR
  }

  return GENERIC_ERROR
}
