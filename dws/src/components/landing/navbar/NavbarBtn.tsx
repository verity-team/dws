import { ReactElement, MouseEvent } from "react";
import { customColors } from "../../../../tailwind.config";

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
  return (
    <button
      type="button"
      className="text-2xl tracking-wide no-underline uppercase hover:text-cred"
      style={{ color: isActive ? customColors.cred : customColors.cblack }}
      onClick={onClick}
    >
      {text}
    </button>
  );
};

export default NavbarButton;
