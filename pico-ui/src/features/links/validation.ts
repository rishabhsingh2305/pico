import { tagValidationError } from "@/lib/tags"
import { byteLength } from "@/lib/text"

const URL_MAX_BYTES = 2048
const TITLE_MAX_BYTES = 120
const SLUG_MIN_LENGTH = 3
const SLUG_MAX_LENGTH = 32
const SLUG_PATTERN = /^[a-z0-9](?:[a-z0-9-]{1,30}[a-z0-9])$/

export type LinkFormErrors = {
  url?: string
  slug?: string
  title?: string
  tags?: string
}

export type LinkFormValues = LinkDetailsValues & {
  slug: string
}

export type LinkDetailsValues = {
  url: string
  title: string
  tags: string[]
}

export function validateLinkURL(raw: string): string | undefined {
  const value = raw.trim()
  if (value === "") {
    return "Enter a destination URL."
  }
  if (byteLength(value) > URL_MAX_BYTES) {
    return "URL must be at most 2048 characters."
  }

  let parsed: URL
  try {
    parsed = new URL(value)
  } catch {
    return "Enter a valid http or https URL."
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    return "Only http and https URLs are supported."
  }
  if (parsed.hostname === "") {
    return "Enter a valid http or https URL."
  }
  if (parsed.username !== "" || parsed.password !== "") {
    return "URLs with credentials are not supported."
  }
  return undefined
}

export function validateLinkSlug(raw: string): string | undefined {
  const slug = raw.trim().toLowerCase()
  if (slug === "") {
    return undefined
  }
  if (slug.length < SLUG_MIN_LENGTH || slug.length > SLUG_MAX_LENGTH) {
    return "Alias must be 3 to 32 characters."
  }
  if (!SLUG_PATTERN.test(slug)) {
    return "Use lowercase letters, numbers, and hyphens; start and end with a letter or number."
  }
  return undefined
}

export function validateLinkTitle(raw: string): string | undefined {
  if (byteLength(raw.trim()) > TITLE_MAX_BYTES) {
    return "Title must be at most 120 characters."
  }
  return undefined
}

export function validateLinkDetails(values: LinkDetailsValues): LinkFormErrors {
  const errors: LinkFormErrors = {}
  const urlError = validateLinkURL(values.url)
  const titleError = validateLinkTitle(values.title)
  const tagsError = tagValidationError(values.tags)

  if (urlError) {
    errors.url = urlError
  }
  if (titleError) {
    errors.title = titleError
  }
  if (tagsError) {
    errors.tags = tagsError
  }
  return errors
}

export function validateLinkForm(values: LinkFormValues): LinkFormErrors {
  const errors = validateLinkDetails(values)
  const slugError = validateLinkSlug(values.slug)
  if (slugError) {
    errors.slug = slugError
  }
  return errors
}
