import { useEffect, useRef, useState } from "react"

type CountUpProps = {
  value: number
  format: (value: number) => string
  className?: string
  durationMs?: number
}

export function CountUp({
  value,
  format,
  className,
  durationMs = 700,
}: CountUpProps) {
  const [displayedValue, setDisplayedValue] = useState(0)
  const displayedValueRef = useRef(0)

  useEffect(() => {
    const from = displayedValueRef.current
    if (from === value) {
      return
    }

    const duration = prefersReducedMotion() ? 0 : durationMs
    const startedAt = Date.now()
    let frameId = 0
    let cancelled = false

    function update(nextValue: number) {
      displayedValueRef.current = nextValue
      setDisplayedValue(nextValue)
    }

    function tick() {
      if (cancelled) {
        return
      }

      const elapsed = Date.now() - startedAt
      const progress =
        duration === 0 ? 1 : Math.min(1, Math.max(0, elapsed / duration))
      const eased = 1 - Math.pow(1 - progress, 3)
      const nextValue = Math.round(from + (value - from) * eased)

      update(nextValue)

      if (progress < 1) {
        frameId = requestFrame(tick)
      }
    }

    frameId = requestFrame(tick)

    return () => {
      cancelled = true
      cancelFrame(frameId)
    }
  }, [durationMs, value])

  return <span className={className}>{format(displayedValue)}</span>
}

function prefersReducedMotion(): boolean {
  return (
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  )
}

function requestFrame(callback: () => void): number {
  if (typeof requestAnimationFrame === "function") {
    return requestAnimationFrame(callback)
  }

  return window.setTimeout(callback, 16)
}

function cancelFrame(frameId: number): void {
  if (typeof cancelAnimationFrame === "function") {
    cancelAnimationFrame(frameId)
    return
  }

  window.clearTimeout(frameId)
}
