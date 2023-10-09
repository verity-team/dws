import React, { ReactNode, ReactElement } from "react";

interface TextErrorProps {
  children?: ReactNode;
}

const TextError = ({
  children,
}: TextErrorProps): ReactElement<TextErrorProps> => {
  return <div className="italic text-sm text-red-500">{children}</div>;
};
export default TextError;
