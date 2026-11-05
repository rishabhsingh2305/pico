import { ApiError } from "@/lib/api-client"

const GENERIC_ERROR = "Something went wrong. Please try again."

export function projectErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 400) {
      return error.message
    }
    if (error.status === 403) {
      return "You do not have permission to change this project."
    }
    if (error.status === 404) {
      return "The project or organization is no longer available."
    }
    return GENERIC_ERROR
  }

  if (error instanceof TypeError) {
    return "Could not reach the server. Check your connection and try again."
  }

  return GENERIC_ERROR
}
