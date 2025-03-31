import { createContext } from "react";

type BlockContextValue = {
  isActive: boolean;
};
const blockContext = createContext<BlockContextValue>({
  isActive: false,
});

type BlockProviderProps = React.PropsWithChildren<{
  isActive: boolean;
}>;
const BlockProvider: React.FC<BlockProviderProps> = ({ isActive, children }) => {
  return <blockContext.Provider value={{ isActive }}>{children}</blockContext.Provider>;
};

export { blockContext, BlockProvider };
