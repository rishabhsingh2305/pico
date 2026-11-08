import type { ClipboardEvent, KeyboardEvent } from "react"
import { XIcon } from "lucide-react"

import { cn } from "@/lib/utils"

type TagInputProps = {
  id: string
  tags: string[]
  input: string
  onInputChange: (value: string) => void
  onAdd: (value: string) => void
  onRemove: (tag: string) => void
  error?: string
  describedBy?: string
  disabled?: boolean
}

export function TagInput({
  id,
  tags,
  input,
  onInputChange,
  onAdd,
  onRemove,
  error,
  describedBy,
  disabled = false,
}: TagInputProps) {
  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter") {
      if (input.trim() !== "") {
        event.preventDefault()
        onAdd(input)
      }
      return
    }

    if (event.key === " " || event.key === ",") {
      event.preventDefault()
      if (input.trim() !== "") {
        onAdd(input)
      }
      return
    }

    if (event.key === "Backspace" && input === "" && tags.length > 0) {
      event.preventDefault()
      onRemove(tags[tags.length - 1])
    }
  }

  function handlePaste(event: ClipboardEvent<HTMLInputElement>) {
    const text = event.clipboardData.getData("text")
    if (/[\s,]/.test(text)) {
      event.preventDefault()
      onAdd(text)
    }
  }

  return (
    <div
      className={cn(
        "flex min-h-11 flex-wrap items-center gap-1.5 border-2 bg-background px-2 py-1.5 transition-colors focus-within:ring-2 focus-within:ring-primary focus-within:ring-offset-2 focus-within:ring-offset-background",
        error ? "border-destructive" : "border-foreground",
        disabled && "opacity-60"
      )}
    >
      {tags.map((tag) => (
        <span
          key={tag}
          className="inline-flex items-center gap-1 border border-foreground/40 bg-muted px-1.5 py-0.5 text-xs font-semibold"
        >
          {tag}
          <button
            type="button"
            onClick={() => onRemove(tag)}
            disabled={disabled}
            aria-label={`Remove ${tag}`}
            className="p-0.5 text-muted-foreground hover:text-foreground disabled:pointer-events-none"
          >
            <XIcon className="size-3.5" aria-hidden="true" />
          </button>
        </span>
      ))}

      <input
        id={id}
        name="tags"
        value={input}
        onChange={(event) => onInputChange(event.target.value)}
        onKeyDown={handleKeyDown}
        onPaste={handlePaste}
        placeholder={tags.length === 0 ? "food, travel" : ""}
        disabled={disabled}
        className="min-w-24 flex-1 bg-transparent py-0.5 text-sm outline-none placeholder:text-muted-foreground/70 disabled:cursor-not-allowed"
        aria-invalid={Boolean(error)}
        aria-describedby={describedBy}
      />
    </div>
  )
}
