import { useCallback, useEffect, useRef, type RefObject } from "react"

type DeviceOrientationEventConstructor = typeof DeviceOrientationEvent & {
  requestPermission?: () => Promise<"granted" | "denied">
}

const MAX_SENSOR_DELTA = 20
const MAX_CARD_TILT = 6
const SMOOTHING = 0.18
const EPSILON = 0.01

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

export function useDeviceTilt(targetRef: RefObject<HTMLElement | null>) {
  const supported =
    typeof window !== "undefined" &&
    "DeviceOrientationEvent" in window &&
    window.matchMedia("(pointer: coarse)").matches &&
    !window.matchMedia("(prefers-reduced-motion: reduce)").matches

  const activeRef = useRef(false)
  const baselineRef = useRef<{ beta: number; gamma: number } | null>(null)
  const smoothedRef = useRef({ x: 0, y: 0 })
  const handlerRef = useRef<((event: DeviceOrientationEvent) => void) | null>(
    null
  )

  const applyTilt = useCallback(() => {
    const element = targetRef.current
    if (!element) {
      return
    }

    element.style.setProperty("--tilt-x", `${smoothedRef.current.x}deg`)
    element.style.setProperty("--tilt-y", `${smoothedRef.current.y}deg`)
  }, [targetRef])

  const resetTilt = useCallback(() => {
    const element = targetRef.current
    if (!element) {
      return
    }
    element.style.removeProperty("--tilt-x")
    element.style.removeProperty("--tilt-y")
  }, [targetRef])

  const handleOrientation = useCallback(
    (event: DeviceOrientationEvent) => {
      if (event.beta === null || event.gamma === null) {
        return
      }

      if (!baselineRef.current) {
        baselineRef.current = { beta: event.beta, gamma: event.gamma }
        return
      }

      const betaDelta = clamp(
        event.beta - baselineRef.current.beta,
        -MAX_SENSOR_DELTA,
        MAX_SENSOR_DELTA
      )
      const gammaDelta = clamp(
        event.gamma - baselineRef.current.gamma,
        -MAX_SENSOR_DELTA,
        MAX_SENSOR_DELTA
      )

      const targetX = (betaDelta / MAX_SENSOR_DELTA) * MAX_CARD_TILT
      const targetY = (gammaDelta / MAX_SENSOR_DELTA) * MAX_CARD_TILT

      smoothedRef.current.x += (targetX - smoothedRef.current.x) * SMOOTHING
      smoothedRef.current.y += (targetY - smoothedRef.current.y) * SMOOTHING

      if (
        Math.abs(smoothedRef.current.x) < EPSILON &&
        Math.abs(smoothedRef.current.y) < EPSILON
      ) {
        smoothedRef.current = { x: 0, y: 0 }
      }
      applyTilt()
    },
    [applyTilt]
  )

  const detach = useCallback(() => {
    if (handlerRef.current) {
      window.removeEventListener("deviceorientation", handlerRef.current)
      handlerRef.current = null
    }
    activeRef.current = false
    baselineRef.current = null
    smoothedRef.current = { x: 0, y: 0 }
    resetTilt()
  }, [resetTilt])

  const enable = useCallback(async () => {
    if (!supported || activeRef.current) {
      return
    }

    const constructor = window.DeviceOrientationEvent as
      DeviceOrientationEventConstructor | undefined

    try {
      if (constructor?.requestPermission) {
        const permission = await constructor.requestPermission()
        if (permission !== "granted") {
          return
        }
      }

      if (!handlerRef.current) {
        handlerRef.current = handleOrientation
        window.addEventListener("deviceorientation", handlerRef.current)
      }
      activeRef.current = true
    } catch {
      // Motion remains off if permission is unavailable or denied.
    }
  }, [handleOrientation, supported])

  useEffect(() => {
    if (!supported) {
      return
    }

    const constructor = window.DeviceOrientationEvent as
      DeviceOrientationEventConstructor | undefined

    if (!constructor?.requestPermission) {
      void enable()
      return
    }

    // iOS requires permission from a user gesture. Start silently on the first
    // touch anywhere in the app, without adding a visible control.
    const onFirstInteraction = () => {
      void enable()
    }
    document.addEventListener("pointerdown", onFirstInteraction, {
      once: true,
      passive: true,
    })

    return () => {
      document.removeEventListener("pointerdown", onFirstInteraction)
    }
  }, [enable, supported])

  useEffect(() => {
    return () => {
      detach()
    }
  }, [detach])
}
