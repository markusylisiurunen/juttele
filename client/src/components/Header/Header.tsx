import { Menu, RefreshCcw, Trash } from "lucide-react";
import React from "react";
import { useApp } from "../../hooks";
import { useAtomWithSelector } from "../../utils";
import styles from "./Header.module.css";

type HeaderProps = {
  chatId: number;
  onNewChat: () => void;
  onRenameChat: () => void;
  onShowChats: () => void;
};
const Header: React.FC<HeaderProps> = ({ chatId, onNewChat, onRenameChat, onShowChats }) => {
  const title = useAtomWithSelector(
    useApp().data,
    (state) => state.chats.find((chat) => chat.id === chatId)?.title ?? ""
  );
  return (
    <div className={styles.root}>
      <div className={styles.container}>
        <div className={styles.left}>
          <button
            className={styles.iconButton}
            style={{ marginInlineStart: "-8px" }}
            onClick={onShowChats}
          >
            <Menu size={15} strokeWidth={2} />
          </button>
          <h1>{title}</h1>
          <button className={styles.iconButton} onClick={onRenameChat}>
            <RefreshCcw size={15} strokeWidth={2} />
          </button>
        </div>
        <div className={styles.right}>
          <button className={styles.iconButton}>
            <Trash size={15} strokeWidth={2} />
          </button>
          <button className={styles.textButton} onClick={onNewChat}>
            <span>New chat</span>
          </button>
        </div>
      </div>
    </div>
  );
};

export { Header };
