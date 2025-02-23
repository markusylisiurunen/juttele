import React from "react";

type TableProps = React.PropsWithChildren<unknown>;
const Table: React.FC<TableProps> = ({ children }) => {
  return (
    <div>
      <table>{children}</table>
    </div>
  );
};

export { Table };
