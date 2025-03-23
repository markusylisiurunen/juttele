import classNames from "classnames";
import React from "react";
import styles from "./Button.module.css";

type ButtonProps = {
  className?: string;
  glowing?: boolean;
  iconLeft?: React.ReactNode;
  iconRight?: React.ReactNode;
  label: string;
  onClick?: () => void;
};
const Button: React.FC<ButtonProps> = ({
  className,
  glowing,
  iconLeft,
  iconRight,
  label,
  onClick,
}) => {
  return (
    <button
      data-glowing={glowing ? "" : undefined}
      className={classNames(styles.button, glowing ? styles.glowing : null, className)}
      onClick={onClick}
    >
      {iconLeft}
      <span>{label}</span>
      {iconRight}
    </button>
  );
};

export { Button };
