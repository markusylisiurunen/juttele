import React from "react";
import { DataResponse } from "../../api";
import { Atom, useAtomWithSelector } from "../../utils";
import { IconButton } from "../IconButton/IconButton";
import styles from "./Header.module.css";

type HeaderProps = {
  dataAtom: Atom<DataResponse>;
  chatId: number;
  onCopyChat: () => void;
  onNewChat: () => void;
  onRenameChat: () => void;
  onShowChats: () => void;
};
const Header: React.FC<HeaderProps> = ({
  dataAtom,
  chatId,
  onCopyChat,
  onNewChat,
  onRenameChat,
  onShowChats,
}) => {
  const title = useAtomWithSelector(
    dataAtom,
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
