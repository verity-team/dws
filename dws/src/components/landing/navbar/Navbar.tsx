"use client";

import Image from "next/image";
import { ReactElement, useLayoutEffect, useState } from "react";
import NavbarItem from "./NavbarItem";
import NavbarDropdown from "./dropdown/NavbarDropdown";
import ConnectButton from "./ConnectBtn";
import { NavbarDropdownItemProps } from "./dropdown/NavbarDropdownItem";
import { Collapse } from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";
import CollapseNavbarDropdown from "./dropdown/CollapseNavbarDropdown";

const aboutOptions: NavbarDropdownItemProps[] = [
  { text: "Whitepaper", href: "/" },
  { text: "Airdrop", href: "/" },
  { text: "Contact", href: "/" },
];

const Navbar = (): ReactElement => {
  const [isMenuOpen, setMenuOpen] = useState(false);

  // Avoid showing 2 navbar at the same time
  useLayoutEffect(() => {
    const closeCollapseMenu = () => {
      setMenuOpen(false);
    };

    window.addEventListener("resize", closeCollapseMenu);

    return () => window.removeEventListener("resize", closeCollapseMenu);
  }, []);

  const handleMenuToggle = () => {
    setMenuOpen((isOpen) => !isOpen);
  };

  const handleMenuClose = () => {
    setMenuOpen(false);
  };

  return (
    <div className="p-5 mx-auto bg-white z-auto">
      <div className="px-5 items-center flex justify-between">
        <div>
          <Image
            src="/images/logo-text.png"
            alt="truth memes logo"
            width={388}
            height={78}
          />
        </div>
        <nav className="hidden items-center space-x-2 lg:flex">
          <NavbarItem text="home" isActive={true} href="#" />
          <NavbarItem text="community" href="#" />
          <NavbarItem text="staking" href="#" />
          <NavbarItem text="memes" href="#" />
          <NavbarDropdown
            title="about"
            options={aboutOptions}
            openBehavior="hover"
          />
          <div className="px-4">
            <ConnectButton />
          </div>
        </nav>
        <div className="flex items-center lg:hidden">
          <button type="button" onClick={handleMenuToggle}>
            <MenuIcon fontSize="large" />
          </button>
        </div>
      </div>
      <Collapse in={isMenuOpen} timeout="auto" unmountOnExit>
        <nav className="flex flex-col items-center justify-center space-x-2 mt-4 md:flex-col md:space-y-2">
          <NavbarItem text="home" isActive={true} href="#" />
          <NavbarItem text="community" href="#" />
          <NavbarItem text="staking" href="#" />
          <NavbarItem text="memes" href="#" />
          <CollapseNavbarDropdown title="About" options={aboutOptions} />
          <div className="pt-4 px-4 md:mt-0">
            <ConnectButton />
          </div>
        </nav>
      </Collapse>
    </div>
  );
};

export default Navbar;
