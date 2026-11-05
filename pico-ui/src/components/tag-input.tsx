import { useState, type ChangeEvent, type KeyboardEvent } from "react"
import { XIcon } from "lucide-react"
import { cn } from "cn"

import { Badge } from "@/components/ui/badge"

type TagInputProps = {
  id: string
  name: string
  value: string[]
  onChange: (tags: string[]) => void
  placeholder?: string
  disabled?: boolean
  invalid?: boolean
  describedBy?: string
}

export function TagInput({
  id,
  name,
  value,
  onChange,
  placeholder,
  disabled = false,
  invalid = false,
  describedBy,
}: TagInputProps) {
  const [draft, setDraft] = useState("")

  function addTags(rawTags: string[], nextDraft: string) {
    const next = [...value]
    for (const rawTag of rawTags) {
      const tag = rawTag.trim()
      if (tag !== "" && !next.includes(tag)) {
        next.push(tag)
      }
    }
    if (next.length !== value.length) {
      onChange(next)
    }
    setDraft(nextDraft)
  }

  function commitDraft() {
    if (draft.trim() !== "") {
      addTags([draft], "")
    }
  }

  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const raw = event.target.value
    if (!raw.includes(",")) {
      setDraft(raw)
      return
    }

    const parts = raw.split(",")
    addTags(parts.slice(0, -1), parts.at(-1) ?? "")
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === ",") {
      event.preventDefault()
      commitDraft()
      return
    }
    if (event.key === "Enter" && draft.trim() !== "") {
      event.preventDefault()
      commitDraft()
      return
    }
    if (event.key === "Backspace" && draft === "" && value.length > 0) {
      event.preventDefault()
      onChange(value.slice(0, -1))
    }
  }

  function removeTag(tag: string) {
    onChange(value.filter((item) => item !== tag))
  }

  return (
    <div
      data-disabled={disabled}
      data-invalid={invalid}
      className={cn(
        "flex min-h-8 w-full flex-wrap items-center gap-1 rounded-lg border border-input bg-transparent px-1.5 py-1 text-base transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 data-[disabled=true]:pointer-events-none data-[disabled=true]:bg-input/50 data-[disabled=true]:opacity-50 data-[invalid=true]:border-destructive data-[invalid=true]:ring-3 data-[invalid=true]:ring-destructive/20 md:text-sm dark:bg-input/30 dark:data-[invalid=true]:border-destructive/50 dark:data-[invalid=true]:ring-destructive/40"
      )}
    >
      {value.map((tag) => (
        <Badge key={tag} variant="secondary" className="gap-0.5 pr-0.5">
          {tag}
          <button
            type="button"
            onClick={() => removeTag(tag)}
            onMouseDown={(event) => event.preventDefault()}
            disabled={disabled}
            aria-label={`Remove tag ${tag}`}
            className="rounded-full p-0.5 text-muted-foreground transition-colors hover:bg-foreground/10 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <XIcon aria-hidden="true" />
          </button>
        </Badge>
      ))}

      <input
        id={id}
        name={name}
        type="text"
        value={draft}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        onBlur={commitDraft}
        placeholder={value.length === 0 ? placeholder : undefined}
        disabled={disabled}
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        autoComplete="off"
        className="h-6 min-w-20 flex-1 border-0 bg-transparent px-1 text-base outline-none placeholder:text-muted-foreground md:text-sm"
      />
    </div>
  )
}
