import classNames from "classnames";
import { DynamicIcon, IconName } from "lucide-react/dynamic";
import React from "react";
import styles from "./Button.module.css";

type ButtonProps = {
  glowing?: boolean;
  icon?: IconName;
  label: string;
  onClick?: () => void;
};
const Button: React.FC<ButtonProps> = ({ glowing, icon, label, onClick }) => {
  return (
    <button
      className={classNames(styles.button, glowing ? styles.glowing : null)}
      onClick={onClick}
    >
      {icon ? <DynamicIcon name={icon} size={15} /> : null}
      <span>{label}</span>
    </button>
  );
};

export { Button };
