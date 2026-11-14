const BLOCKS = [
  {
    background: "var(--block-1)",
    foreground: "var(--block-1-foreground)",
    icon: "var(--block-1-icon)",
    border: "var(--block-1-border)",
  },
  {
    background: "var(--block-2)",
    foreground: "var(--block-2-foreground)",
    icon: "var(--block-2-icon)",
    border: "var(--block-2-border)",
  },
  {
    background: "var(--block-3)",
    foreground: "var(--block-3-foreground)",
    icon: "var(--block-3-icon)",
    border: "var(--block-3-border)",
  },
  {
    background: "var(--block-4)",
    foreground: "var(--block-4-foreground)",
    icon: "var(--block-4-icon)",
    border: "var(--block-4-border)",
  },
  {
    background: "var(--block-5)",
    foreground: "var(--block-5-foreground)",
    icon: "var(--block-5-icon)",
    border: "var(--block-5-border)",
  },
  {
    background: "var(--block-6)",
    foreground: "var(--block-6-foreground)",
    icon: "var(--block-6-icon)",
    border: "var(--block-6-border)",
  },
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
