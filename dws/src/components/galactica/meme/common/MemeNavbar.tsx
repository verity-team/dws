"use client";

import Image from "next/image";
import {
  ReactElement,
  useContext,
  useLayoutEffect,
  useReducer,
  useRef,
  useState,
} from "react";
import { Collapse, Menu, MenuItem } from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";
import NavbarItem from "@/components/landing/navbar/NavbarItem";
import SignInBtn from "../../account/SignInBtn";
import Avatar from "boring-avatars";
import { Wallet } from "@/components/ClientRoot";

const navbarItems = [
  {
    text: "Home",
    isActive: false,
    href: "/",
  },
  {
    text: "Story",
    isActive: false,
    href: "/story.html",
  },
  {
    text: "Community",
    isActive: false,
    href: "/community.html",
  },
  {
    text: "Memes",
    isActive: true,
    href: "#",
  },
  {
    text: "NFTs",
    isActive: false,
    href: "/nft.html",
  },
];

const MemeNavbar = (): ReactElement => {
  const userWallet = useContext(Wallet);
  const [isMenuOpen, setMenuOpen] = useState(false);

  const anchorElement = useRef(null);

  // Avoid showing 2 navbar at the same time
  useLayoutEffect(() => {
    const closeCollapseMenu = () => {
      setMenuOpen(false);
    };

    window.addEventListener("resize", closeCollapseMenu);

    return () => window.removeEventListener("resize", closeCollapseMenu);
  }, []);

  const handleMenuOpen = () => {
    setMenuOpen(true);
  };

  const handleMenuClose = () => {
    setMenuOpen(false);
  };

  return (
    <div className="z-auto md:max-w-6xl md:mx-auto">
      <div className="px-5 items-center flex justify-between">
        {/* LOGO */}
        <a href="/">
          <Image
            src="/dws-images/logo-no-shadow.png"
            alt="truth memes logo"
            width={64}
            height={0}
            className="max-w-[200px]"
          />
        </a>
        <div className="flex items-center space-x-4">
          <SignInBtn />
          <div className="flex items-center space-x-4">
            <button type="button" onClick={handleMenuOpen} ref={anchorElement}>
              <MenuIcon className="w-12 h-12" />
            </button>
          </div>
        </div>
      </div>
      <Menu
        anchorEl={anchorElement.current}
        open={isMenuOpen}
        onClose={handleMenuClose}
        anchorOrigin={{ vertical: "bottom", horizontal: "left" }}
      >
        {navbarItems.map((item) => (
          <MenuItem key={item.text} className="font-changa">
            <NavbarItem
              text={item.text}
              isActive={item.isActive}
              href={item.href}
            />
          </MenuItem>
        ))}
      </Menu>
    </div>
  );
};

export default MemeNavbar;
