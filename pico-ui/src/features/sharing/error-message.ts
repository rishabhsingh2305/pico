import { ApiError } from "@/lib/api-client"

export function shareErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return "You do not have permission to manage sharing for this organization."
    }
    if (error.status === 404) {
      return "The share link or resource no longer exists."
    }
    return error.message
  }
  if (error instanceof TypeError) {
    return "Could not reach the server. Check your connection and try again."
  }
  return "Could not update sharing."
}
