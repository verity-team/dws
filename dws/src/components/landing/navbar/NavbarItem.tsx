import { ReactElement } from "react";
import Link from "next/link";
import NavbarButton from "./NavbarBtn";

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
      <NavbarButton text={text} isActive={isActive} />
    </Link>
  );
};

export default NavbarItem;
