import { ReactElement } from "react";
import config, { customColors } from "../../../../tailwind.config";
import Link from "next/link";

interface NavbarItemProps {
  text: string;
  href: string;
  isActive?: boolean;
}

const NavbarItem = ({
  text,
  href,
  isActive = false,
}: NavbarItemProps): ReactElement<NavbarItemProps> => {
  return (
    <Link className="px-4 py-2 cursor-pointer" href={href}>
      <button
        type="button"
        className="text-2xl tracking-wide no-underline uppercase hover:text-cred"
        style={{ color: isActive ? customColors.cred : customColors.cblack }}
      >
        {text}
      </button>
    </Link>
  );
};

export default NavbarItem;
