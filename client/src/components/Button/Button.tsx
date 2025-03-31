import classNames from "classnames";
import React from "react";
import styles from "./Button.module.css";

type _ButtonProps = {
  className?: string;
  glowing?: boolean;
  iconLeft?: React.ReactNode;
  iconRight?: React.ReactNode;
  label?: string;
  onClick?: () => void;
};
type ButtonProps = Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, keyof _ButtonProps> &
  _ButtonProps;
const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, glowing, iconLeft, iconRight, label, onClick, ...props }, ref) => {
    return (
      <button
        {...props}
        ref={ref}
        data-glowing={glowing ? "" : undefined}
        className={classNames(styles.button, glowing ? styles.glowing : null, className)}
        onClick={onClick}
      >
        {iconLeft}
        {label ? <span>{label}</span> : null}
        {iconRight}
      </button>
    );
  }
);
Button.displayName = "Button";

export { Button };
