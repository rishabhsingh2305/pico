import {
  BabyIcon,
  BriefcaseBusinessIcon,
  CarIcon,
  GiftIcon,
  GraduationCapIcon,
  HeartPulseIcon,
  HouseIcon,
  PawPrintIcon,
  ReceiptTextIcon,
  ShirtIcon,
  ShoppingBasketIcon,
  SmartphoneIcon,
  UserRoundIcon,
  UtensilsCrossedIcon,
  WalletIcon,
  WrenchIcon,
  type LucideIcon,
} from "lucide-react"
import { createElement } from "react"

const KEYWORD_ICONS: Array<[RegExp, LucideIcon]> = [
  [/travel|trip|tour|flight|holiday|vacation/i, BriefcaseBusinessIcon],
  [/bill|rent|utility|electric|water|gas|internet|broadband/i, ReceiptTextIcon],
  [/personal|self/i, UserRoundIcon],
  [
    /food|grocery|groceries|dining|restaurant|lunch|dinner|snack/i,
    ShoppingBasketIcon,
  ],
  [/renovation|repair|construction|home|house/i, HouseIcon],
  [/transport|car|fuel|petrol|taxi|cab|bus|train|metro/i, CarIcon],
  [/health|medical|doctor|medicine|hospital|clinic/i, HeartPulseIcon],
  [/education|school|college|course|book|tuition/i, GraduationCapIcon],
  [/gift|donation|present/i, GiftIcon],
  [/pet|dog|cat/i, PawPrintIcon],
  [/baby|child|kid/i, BabyIcon],
  [/cloth|shirt|fashion|shopping/i, ShirtIcon],
  [/phone|mobile|recharge|sim/i, SmartphoneIcon],
  [/kitchen|cook|utensil/i, UtensilsCrossedIcon],
  [/maintenance|tool|service/i, WrenchIcon],
]

const FALLBACK_ICONS = [
  WalletIcon,
  ReceiptTextIcon,
  ShoppingBasketIcon,
  HouseIcon,
  UserRoundIcon,
  CarIcon,
]

export function categoryIcon(name: string, id: string): LucideIcon {
  for (const [pattern, icon] of KEYWORD_ICONS) {
    if (pattern.test(name)) {
      return icon
    }
  }

  let hash = 2166136261
  for (let index = 0; index < id.length; index += 1) {
    hash ^= id.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return FALLBACK_ICONS[(hash >>> 0) % FALLBACK_ICONS.length]
}

type CategoryGlyphProps = {
  name: string
  id: string
  className?: string
}

export function CategoryGlyph({ name, id, className }: CategoryGlyphProps) {
  return createElement(categoryIcon(name, id), {
    className,
    "aria-hidden": true,
  })
}
