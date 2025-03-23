import React from "react";
import { ErrorBlock } from "../../blocks";
import styles from "./ErrorBlock.module.css";

type ErrorComponentProps = {
  block: ErrorBlock;
};
const ErrorComponent: React.FC<ErrorComponentProps> = ({ block }) => {
  return (
    <div className={styles.root} data-block="error">
      <div className={styles.header}>
        <span>An error happened</span>
      </div>
      <div className={styles.content}>
        <span>{block.error.message}</span>
      </div>
    </div>
  );
};
const MemoedErrorComponent = React.memo(ErrorComponent, (prev, next) => {
  if (prev.block.id !== next.block.id) return false;
  if (prev.block.hash !== next.block.hash) return false;
  return true;
});

export { MemoedErrorComponent as Error };
