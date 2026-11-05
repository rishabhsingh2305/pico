type SectionHeadingProps = {
  id: string
  index: string
  eyebrow: string
  title: string
  meta?: string
}

export function SectionHeading({
  id,
  index,
  eyebrow,
  title,
  meta,
}: SectionHeadingProps) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3">
      <div>
        <div className="flex items-center gap-2">
          <span
            className="font-mono text-[10px] text-muted-foreground tabular-nums"
            aria-hidden="true"
          >
            {index}
          </span>
          <p className="text-label text-primary uppercase">{eyebrow}</p>
        </div>
        <h2 id={id} className="mt-1 font-display text-heading">
          {title}
        </h2>
      </div>
      {meta ? (
        <p className="font-mono text-[10px] tracking-[0.14em] text-muted-foreground uppercase">
          {meta}
        </p>
      ) : null}
    </div>
  )
}
