"use client";

import { Menu, MenuItem, useMediaQuery } from "@mui/material";
import { ReactElement, MouseEvent, useState, useMemo } from "react";
import NavbarButton from "./NavbarBtn";
import { Nullable } from "@/utils/types";
import NavbarItem from "./NavbarItem";

export interface NavbarDropdownItemOption {
  text: string;
  href: string;
}

interface NavbarDropdownItemProps {
  title: string;
  options: NavbarDropdownItemOption[];
}

const NavbarDropdownItem = ({
  title,
  options,
}: NavbarDropdownItemProps): ReactElement<NavbarDropdownItemProps> => {
  const [anchor, setAnchor] = useState<Nullable<HTMLElement>>(null);

  const isOpen = useMemo(() => Boolean(anchor), [anchor]);

  const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
    setAnchor(event.currentTarget);
  };

  const handleClose = () => {
    setAnchor(null);
  };

  return (
    <div>
      <NavbarButton text={title} isActive={false} onClick={handleClick} />
      <Menu anchorEl={anchor} open={isOpen} onClose={handleClose}>
        {options.map(({ text, href }) => (
          <MenuItem key={text}>
            <NavbarItem text={text} href={href} />
          </MenuItem>
        ))}
      </Menu>
    </div>
  );
};

export default NavbarDropdownItem;
