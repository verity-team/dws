import { MouseEventHandler, ReactElement, ReactNode, memo } from "react";

interface TextButtonProps {
  onClick: MouseEventHandler<HTMLButtonElement>;
  children: ReactNode;
}

const TextButton = ({
  children,
  onClick,
}: TextButtonProps): ReactElement<TextButtonProps> => {
  return (
    <button
      type="button"
      className="px-4 py-2 rounded-lg border-2 border-black text-xl"
      onClick={onClick}
    >
      {children}
    </button>
  );
};

export default memo(TextButton);
