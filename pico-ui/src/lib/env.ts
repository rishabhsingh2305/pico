const DEFAULT_API_BASE_URL = "/api"

function normalizeBaseURL(value: string): string {
  return value.replace(/\/+$/, "")
}

export const API_BASE_URL = normalizeBaseURL(
  import.meta.env.VITE_API_BASE_URL?.trim() || DEFAULT_API_BASE_URL
)
