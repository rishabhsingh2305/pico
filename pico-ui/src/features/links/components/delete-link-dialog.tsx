import { LinkIcon } from "lucide-react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Spinner } from "@/components/ui/spinner"
import type { ShortLink } from "@/features/links/types"

type DeleteLinkDialogProps = {
  link: ShortLink | null
  deleting: boolean
  onCancel: () => void
  onConfirm: () => void
}

export function DeleteLinkDialog({
  link,
  deleting,
  onCancel,
  onConfirm,
}: DeleteLinkDialogProps) {
  return (
    <AlertDialog
      open={link !== null}
      onOpenChange={(open) => {
        if (!open && !deleting) {
          onCancel()
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia>
            <LinkIcon aria-hidden="true" />
          </AlertDialogMedia>
          <AlertDialogTitle>Delete link?</AlertDialogTitle>
          <AlertDialogDescription>
            This permanently stops{" "}
            <span className="font-mono text-foreground">/s/{link?.slug}</span>{" "}
            and the alias cannot be reused. This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleting}
            onClick={onConfirm}
          >
            {deleting ? (
              <>
                <Spinner data-icon="inline-start" />
                Deleting...
              </>
            ) : (
              "Delete link"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
