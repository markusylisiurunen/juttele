import classNames from "classnames";
import { DynamicIcon, IconName } from "lucide-react/dynamic";
import React from "react";
import styles from "./IconButton.module.css";

type IconButtonProps = {
  faded?: boolean;
  icon: IconName;
  onClick?: () => void;
};
const IconButton: React.FC<IconButtonProps> = ({ faded, icon, onClick }) => {
  return (
    <button className={classNames(styles.button, faded ? styles.faded : null)} onClick={onClick}>
      <DynamicIcon name={icon} size={15} />
    </button>
  );
};

export { IconButton };
