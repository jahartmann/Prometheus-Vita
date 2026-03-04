"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

interface CollapsibleContextValue {
  open: boolean;
  toggle: () => void;
}

const CollapsibleContext = React.createContext<CollapsibleContextValue>({
  open: false,
  toggle: () => {},
});

interface CollapsibleProps {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  defaultOpen?: boolean;
  children: React.ReactNode;
  className?: string;
}

function Collapsible({
  open: controlledOpen,
  onOpenChange,
  defaultOpen = false,
  children,
  className,
}: CollapsibleProps) {
  const [internalOpen, setInternalOpen] = React.useState(defaultOpen);
  const isControlled = controlledOpen !== undefined;
  const open = isControlled ? controlledOpen : internalOpen;

  const toggle = React.useCallback(() => {
    if (isControlled) {
      onOpenChange?.(!open);
    } else {
      setInternalOpen((prev) => {
        onOpenChange?.(!prev);
        return !prev;
      });
    }
  }, [isControlled, open, onOpenChange]);

  return (
    <CollapsibleContext.Provider value={{ open, toggle }}>
      <div className={className}>{children}</div>
    </CollapsibleContext.Provider>
  );
}

const CollapsibleTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & { asChild?: boolean }
>(({ className, children, asChild, onClick, ...props }, ref) => {
  const { toggle } = React.useContext(CollapsibleContext);

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    toggle();
    onClick?.(e);
  };

  if (asChild && React.isValidElement(children)) {
    return React.cloneElement(children as React.ReactElement<{ onClick?: (e: React.MouseEvent) => void; ref?: React.Ref<HTMLButtonElement> }>, {
      onClick: (e: React.MouseEvent) => {
        toggle();
        (children as React.ReactElement<{ onClick?: (e: React.MouseEvent) => void }>).props.onClick?.(e);
      },
      ref,
    });
  }

  return (
    <button
      ref={ref}
      type="button"
      className={cn(className)}
      onClick={handleClick}
      {...props}
    >
      {children}
    </button>
  );
});
CollapsibleTrigger.displayName = "CollapsibleTrigger";

interface CollapsibleContentProps extends React.HTMLAttributes<HTMLDivElement> {}

function CollapsibleContent({ className, children, ...props }: CollapsibleContentProps) {
  const { open } = React.useContext(CollapsibleContext);

  if (!open) return null;

  return (
    <div className={cn(className)} {...props}>
      {children}
    </div>
  );
}

export { Collapsible, CollapsibleTrigger, CollapsibleContent };
