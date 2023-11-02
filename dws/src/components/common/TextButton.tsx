import { MouseEventHandler, ReactElement, ReactNode, memo } from "react";

interface TextButtonProps {
  onClick: MouseEventHandler<HTMLButtonElement>;
  disabled?: boolean;
  children: ReactNode;
}

const TextButton = ({
  children,
  disabled = false,
  onClick,
}: TextButtonProps): ReactElement<TextButtonProps> => {
  return (
    <button
      disabled={disabled}
      type="button"
      className="px-4 py-2 rounded-lg border-2 border-black disabled:border-gray-500 disabled:text-gray-500 hover:bg-gray-100"
      onClick={onClick}
    >
      {children}
    </button>
  );
};

export default memo(TextButton);
