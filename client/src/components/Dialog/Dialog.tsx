import classNames from "classnames";
import { Dialog as RadixDialog, VisuallyHidden } from "radix-ui";
import React from "react";
import styles from "./Dialog.module.css";

type DialogOverlayProps = React.ComponentProps<typeof RadixDialog.Overlay>;
const DialogOverlay = React.forwardRef<HTMLDivElement, DialogOverlayProps>(
  ({ className, ...props }, ref) => (
    <RadixDialog.Overlay ref={ref} className={classNames(styles.overlay, className)} {...props} />
  )
);
DialogOverlay.displayName = "DialogOverlay";

type DialogContentProps = React.ComponentProps<typeof RadixDialog.Content>;
const DialogContent = React.forwardRef<HTMLDivElement, DialogContentProps>(
  ({ className, children, ...props }, ref) => (
    <RadixDialog.Content ref={ref} className={classNames(styles.content, className)} {...props}>
      {children}
    </RadixDialog.Content>
  )
);
DialogContent.displayName = "DialogContent";

type DialogProps = React.PropsWithChildren<{
  title: string;
  trigger: React.ReactNode;
}>;
const Dialog: React.FC<DialogProps> = ({ title, trigger, children }) => {
  return (
    <RadixDialog.Root>
      <RadixDialog.Trigger asChild>{trigger}</RadixDialog.Trigger>
      <RadixDialog.Portal>
        <DialogOverlay />
        <DialogContent>
          <VisuallyHidden.Root>
            <RadixDialog.Title>{title}</RadixDialog.Title>
            <RadixDialog.Description>{title}</RadixDialog.Description>
          </VisuallyHidden.Root>
          {children}
        </DialogContent>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
};

export { Dialog };
