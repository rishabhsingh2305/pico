import { tagValidationError } from "@/lib/tags"
import { byteLength } from "@/lib/text"

const NAME_MAX_BYTES = 120

export function validateProjectName(raw: string): string | undefined {
  const name = raw.trim()
  if (name === "") {
    return "Enter a project name."
  }
  if (byteLength(name) > NAME_MAX_BYTES) {
    return "Name must be at most 120 characters."
  }
  return undefined
}

export function validateProjectTags(tags: string[]): string | undefined {
  return tagValidationError(tags)
}
