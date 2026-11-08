export function byteLength(value: string): number {
  return new TextEncoder().encode(value).length
}
