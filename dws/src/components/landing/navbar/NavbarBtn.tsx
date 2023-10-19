import { ReactElement, MouseEvent, memo } from "react";
import { customColors } from "@root/tailwind.config";

interface NavbarButtonProps {
  text: string;
  isActive: boolean;
  onClick?: (event: MouseEvent<HTMLButtonElement>) => void;
}

const NavbarButton = ({
  text,
  isActive,
  // Do nothing by default
  onClick = undefined,
}: NavbarButtonProps): ReactElement<NavbarButtonProps> => {
  if (isActive) {
    return (
      <button
        type="button"
        className="text-2xl tracking-wide no-underline uppercase text-cblack hover:text-cred"
        style={{ color: customColors.cred }}
        onClick={onClick}
      >
        {text}
      </button>
    );
  }

  return (
    <button
      type="button"
      className="text-2xl tracking-wide no-underline uppercase text-cblack hover:text-cred"
      style={{ color: customColors.cblack }}
      onClick={onClick}
    >
      {text}
    </button>
  );
};

export default memo(NavbarButton);
