import React from "react";
import { useApp } from "../../hooks";
import { useAtomWithSelector } from "../../utils";
import { IconButton } from "../IconButton/IconButton";
import styles from "./Header.module.css";

type HeaderProps = {
  chatId: number;
  onCopyChat: () => void;
  onNewChat: () => void;
  onRenameChat: () => void;
  onShowChats: () => void;
};
const Header: React.FC<HeaderProps> = ({
  chatId,
  onCopyChat,
  onNewChat,
  onRenameChat,
  onShowChats,
}) => {
  const app = useApp();
  const title = useAtomWithSelector(
    app.data,
    (data) => data.chats.find((chat) => chat.id === chatId)?.title
  );
  return (
    <div className={styles.root}>
      <div>
        <IconButton icon="menu" onClick={onShowChats} />
        <span>{title}</span>
        <IconButton faded icon="refresh-ccw" onClick={onRenameChat} />
      </div>
      <div>
        <IconButton icon="copy" onClick={onCopyChat} />
        <IconButton icon="pen-square" onClick={onNewChat} />
      </div>
    </div>
  );
};

export { Header };
