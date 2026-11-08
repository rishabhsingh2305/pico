import { byteLength } from "@/lib/text"

export const MAX_TAGS = 20
export const MAX_TAG_BYTES = 32

export type NormalizeTagsResult =
  { ok: true; tags: string[] } | { ok: false; error: string }

export function normalizeTags(raw: string[]): NormalizeTagsResult {
  const seen = new Set<string>()
  const tags: string[] = []

  for (const value of raw) {
    const tag = value.trim()
    if (tag === "") {
      continue
    }
    if (byteLength(tag) > MAX_TAG_BYTES) {
      return {
        ok: false,
        error: `Tags must be at most ${MAX_TAG_BYTES} characters.`,
      }
    }

    const key = tag.toLowerCase()
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    tags.push(tag)
  }

  if (tags.length > MAX_TAGS) {
    return {
      ok: false,
      error: `You can add up to ${MAX_TAGS} tags.`,
    }
  }

  return { ok: true, tags }
}
