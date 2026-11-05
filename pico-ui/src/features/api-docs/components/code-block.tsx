import { CopyButton } from "@/features/api-docs/components/copy-button"

type CodeBlockProps = {
  code: string
  label: string
}

export function CodeBlock({ code, label }: CodeBlockProps) {
  return (
    <div className="overflow-hidden rounded-lg border bg-muted/40">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-1.5">
        <span className="text-xs font-medium text-muted-foreground">
          {label}
        </span>
        <CopyButton value={code} label={`Copy ${label.toLowerCase()}`} />
      </div>
      <pre className="overflow-x-auto p-3 text-xs leading-relaxed">
        <code>{code}</code>
      </pre>
    </div>
  )
}
