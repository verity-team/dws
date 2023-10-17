import Link from "next/link";
import { ReactElement, memo } from "react";

export interface NavbarDropdownItemProps {
  text: string;
  href: string;
}

const NavbarDropdownItem = ({
  text,
  href,
}: NavbarDropdownItemProps): ReactElement<NavbarDropdownItemProps> => {
  return (
    <Link className="cursor-pointer" href={href}>
      <button
        type="button"
        className="text-lg font-semibold tracking-wide no-underline capitalize text-cblack hover:text-cred"
      >
        {text}
      </button>
    </Link>
  );
};

export default memo(NavbarDropdownItem);
