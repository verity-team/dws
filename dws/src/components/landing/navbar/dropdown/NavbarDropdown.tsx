"use client";

import { Menu, MenuItem, useMediaQuery } from "@mui/material";
import { ReactElement, MouseEvent, useState, useMemo } from "react";
import NavbarButton from "../NavbarBtn";
import { Nullable } from "@/utils/types";
import NavbarItem from "../NavbarItem";
import KeyboardArrowDownIcon from "@mui/icons-material/KeyboardArrowDown";
import NavbarDropdownItem, {
  NavbarDropdownItemProps,
} from "./NavbarDropdownItem";
import { customColors } from "../../../../../tailwind.config";

interface NavbarDropdownProps {
  title: string;
  options: NavbarDropdownItemProps[];
}

const NavbarDropdown = ({
  title,
  options,
}: NavbarDropdownProps): ReactElement<NavbarDropdownProps> => {
  const [anchor, setAnchor] = useState<Nullable<HTMLElement>>(null);

  const isOpen = useMemo(() => anchor !== null, [anchor]);

  const handleClick = (event: MouseEvent<HTMLElement>) => {
    if (anchor !== event.currentTarget) {
      setAnchor(event.currentTarget);
    }
  };

  const handleClose = () => {
    setAnchor(null);
  };

  return (
    <div>
      <div className="flex items-center text-cblack" onMouseOver={handleClick}>
        <button
          type="button"
          className="text-2xl tracking-wide no-underline uppercase"
          onClick={handleClick}
        >
          {title}
        </button>
        <KeyboardArrowDownIcon />
      </div>
      <Menu
        anchorEl={anchor}
        open={isOpen}
        onClose={handleClose}
        autoFocus={false}
        disableAutoFocus={true}
        disableAutoFocusItem={true}
        hideBackdrop={true}
        MenuListProps={{ onMouseLeave: handleClose }}
      >
        {options.map(({ text, href }) => (
          <MenuItem
            key={text}
            style={{ backgroundColor: "transparent" }}
            disableGutters={true}
            className="px-2 py-1"
          >
            <NavbarDropdownItem text={text} href={href} />
          </MenuItem>
        ))}
      </Menu>
    </div>
  );
};

export default NavbarDropdown;
