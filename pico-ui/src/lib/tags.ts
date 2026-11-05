import { byteLength } from "@/lib/text"

const MAX_TAGS = 20
const MAX_TAG_BYTES = 32

export function normalizeTags(tags: string[]): string[] {
  const seen = new Set<string>()
  const normalized: string[] = []

  for (const rawTag of tags) {
    const tag = rawTag.trim()
    if (tag === "" || seen.has(tag)) {
      continue
    }
    seen.add(tag)
    normalized.push(tag)
  }

  return normalized
}

export function tagValidationError(tags: string[]): string | undefined {
  const normalized = normalizeTags(tags)

  if (normalized.length > MAX_TAGS) {
    return "Use at most 20 tags."
  }
  if (normalized.some((tag) => byteLength(tag) > MAX_TAG_BYTES)) {
    return "Each tag must be at most 32 characters."
  }
  return undefined
}
