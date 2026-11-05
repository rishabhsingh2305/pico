import { tagValidationError } from "@/lib/tags"
import { byteLength } from "@/lib/text"

const NAME_MAX_BYTES = 120

export function validateOrgName(raw: string): string | undefined {
  const name = raw.trim()
  if (name === "") {
    return "Enter an organization name."
  }
  if (byteLength(name) > NAME_MAX_BYTES) {
    return "Name must be at most 120 characters."
  }
  return undefined
}

export function validateOrgTags(tags: string[]): string | undefined {
  return tagValidationError(tags)
}
