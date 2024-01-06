"use client";

import Image from "next/image";
import { ReactElement, useLayoutEffect, useState } from "react";
import NavbarItem from "./NavbarItem";
import { Collapse } from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";

const navbarItems = [
  {
    text: "Home",
    isActive: true,
    href: "/",
  },
  {
    text: "Community",
    isActive: false,
    href: "/",
  },
  {
    text: "Staking",
    isActive: false,
    href: "/",
  },
  {
    text: "Memes",
    isActive: false,
    href: "/meme",
  },
  {
    text: "NFTs",
    isActive: false,
    href: "/",
  },
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

  return (
    <div className="mx-8 z-auto md:mx-12">
      <div className="px-5 items-center flex justify-between">
        {/* LOGO */}
        <div>
          <Image
            src="/images/logo-no-shadow.png"
            alt="truth memes logo"
            width={100}
            height={0}
            className="max-w-[200px]"
          />
        </div>

        {/* DESKTOP NAVBAR */}
        <nav className="hidden items-center space-x-2 lg:flex">
          {navbarItems.map((item) => (
            <NavbarItem
              text={item.text}
              isActive={item.isActive}
              href={item.href}
              key={item.text}
            />
          ))}
        </nav>

        {/* MOBILE NAVBAR */}
        <div className="flex items-center lg:hidden">
          <button type="button" onClick={handleMenuToggle}>
            <MenuIcon fontSize="large" />
          </button>
        </div>
      </div>
      <Collapse in={isMenuOpen} timeout="auto" unmountOnExit>
        <nav className="flex flex-col items-center justify-center space-y-1">
          {navbarItems.map((item) => (
            <NavbarItem
              text={item.text}
              isActive={item.isActive}
              href={item.href}
              key={item.text}
            />
          ))}
        </nav>
      </Collapse>
    </div>
  );
};

export default Navbar;
