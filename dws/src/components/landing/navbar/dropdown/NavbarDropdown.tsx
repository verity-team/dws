"use client";

import { Menu, MenuItem } from "@mui/material";
import { ReactElement, MouseEvent, useState, useMemo } from "react";
import { Nullable } from "@/utils/types";
import KeyboardArrowDownIcon from "@mui/icons-material/KeyboardArrowDown";
import NavbarDropdownItem, {
  NavbarDropdownItemProps,
} from "./NavbarDropdownItem";

interface NavbarDropdownProps {
  title: string;
  options: NavbarDropdownItemProps[];
  // Hover mostly for desktop, Click for mobile
  openBehavior: "hover" | "click";
}

const NavbarDropdown = ({
  title,
  options,
  openBehavior,
}: NavbarDropdownProps): ReactElement<NavbarDropdownProps> => {
  const [anchor, setAnchor] = useState<Nullable<HTMLElement>>(null);

  const isOpen = useMemo(() => anchor !== null, [anchor]);

  const handleClick = (event: MouseEvent<HTMLElement>) => {
    if (anchor != null) {
      return;
    }
    setAnchor(event.currentTarget);
  };

  const handleClose = () => {
    setAnchor(null);
  };

  return (
    <>
      <div
        className="flex items-center text-cblack"
        onMouseOver={openBehavior === "hover" ? handleClick : undefined}
        onClick={handleClick}
      >
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
        MenuListProps={{
          onMouseLeave: openBehavior === "hover" ? handleClose : undefined,
        }}
        style={{ boxShadow: "none" }}
        sx={(theme) => ({
          "& .MuiMenu-paper": {
            boxShadow: "none",
          },
        })}
      >
        {options.map(({ text, href }) => (
          <MenuItem
            key={text}
            style={{ backgroundColor: "transparent" }}
            disableGutters={true}
            className="px-2 py-1"
            onClick={handleClose}
          >
            <NavbarDropdownItem text={text} href={href} />
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};

export default NavbarDropdown;
