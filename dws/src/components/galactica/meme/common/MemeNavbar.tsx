"use client";

import Image from "next/image";
import { ReactElement, useContext, useLayoutEffect, useState } from "react";
import { Collapse } from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";
import NavbarItem from "@/components/landing/navbar/NavbarItem";
import SignInBtn from "../../account/SignInBtn";
import Avatar from "boring-avatars";
import { ClientWallet } from "@/components/ClientRoot";

const navbarItems = [
  {
    text: "Home",
    isActive: false,
    href: "/",
  },
  {
    text: "Story",
    isActive: false,
    href: "https://truthmemes.io/story.html",
  },
  {
    text: "Community",
    isActive: false,
    href: "https://truthmemes.io/community.html",
  },
  {
    text: "Memes",
    isActive: true,
    href: "#",
  },
  {
    text: "NFTs",
    isActive: false,
    href: "https://truthmemes.io/nft.html",
  },
];

const MemeNavbar = (): ReactElement => {
  const account = useContext(ClientWallet);
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
    <div className="z-auto md:max-w-6xl md:mx-auto">
      <div className="px-5 items-center flex justify-between">
        {/* LOGO */}
        <a href="/">
          <Image
            src="/images/logo-no-shadow.png"
            alt="truth memes logo"
            width={100}
            height={0}
            className="max-w-[200px]"
          />
        </a>
        <div className="flex items-center space-x-4">
          <SignInBtn />
          <div className="flex items-center space-x-4">
            <button type="button" onClick={handleMenuToggle}>
              {account ? <Avatar size={40} name={account} /> : <MenuIcon />}
            </button>
          </div>
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

export default MemeNavbar;
