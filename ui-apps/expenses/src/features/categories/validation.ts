import { byteLength } from "@/lib/text"

const NAME_MAX_BYTES = 120

export function validateCategoryName(value: string): string | undefined {
  const name = value.trim()
  if (name === "") {
    return "Enter a category name."
  }
  if (byteLength(name) > NAME_MAX_BYTES) {
    return "Name must be at most 120 characters."
  }
  return undefined
}
