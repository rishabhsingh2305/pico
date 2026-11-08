const BLOCKS = [
  { background: "var(--block-1)", foreground: "var(--block-1-foreground)" },
  { background: "var(--block-2)", foreground: "var(--block-2-foreground)" },
  { background: "var(--block-3)", foreground: "var(--block-3-foreground)" },
  { background: "var(--block-4)", foreground: "var(--block-4-foreground)" },
]

export type CategoryColor = (typeof BLOCKS)[number]

export function categoryColor(id: string): CategoryColor {
  let hash = 2166136261
  for (let index = 0; index < id.length; index += 1) {
    hash ^= id.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return BLOCKS[(hash >>> 0) % BLOCKS.length]
}
