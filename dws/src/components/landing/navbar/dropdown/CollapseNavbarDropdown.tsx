import { ReactElement, useCallback, useState } from "react";
import KeyboardArrowDownIcon from "@mui/icons-material/KeyboardArrowDown";
import NavbarDropdownItem, {
  NavbarDropdownItemProps,
} from "./NavbarDropdownItem";
import { Collapse } from "@mui/material";

interface CollapseNavbarDropdownProps {
  title: string;
  options: NavbarDropdownItemProps[];
}

const CollapseNavbarDropdown = ({
  title,
  options,
}: CollapseNavbarDropdownProps): ReactElement<CollapseNavbarDropdownProps> => {
  const [menuOpen, setMenuOpen] = useState(false);

  const handleClick = useCallback(() => {
    setMenuOpen((isOpen) => !isOpen);
  }, []);

  return (
    <>
      <div
        className="flex items-center text-cblack cursor-pointer hover:text-cred"
        onClick={handleClick}
      >
        <button
          type="button"
          className="text-2xl tracking-wide no-underline uppercase"
        >
          {title}
        </button>
        <KeyboardArrowDownIcon />
      </div>
      <Collapse in={menuOpen} timeout="auto" unmountOnExit>
        <div className="flex flex-col space-y-2 items-center">
          {options.map(({ text, href }) => (
            <NavbarDropdownItem key={text} text={text} href={href} />
          ))}
        </div>
      </Collapse>
    </>
  );
};

export default CollapseNavbarDropdown;
