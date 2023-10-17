import Image from "next/image";
import { ReactElement } from "react";
import NavbarItem from "./NavbarItem";
import NavbarDropdown from "./dropdown/NavbarDropdown";
import ConnectButton from "./ConnectBtn";
import { NavbarDropdownItemProps } from "./dropdown/NavbarDropdownItem";

const aboutOptions: NavbarDropdownItemProps[] = [
  { text: "Whitepaper", href: "/" },
  { text: "Airdrop", href: "/" },
  { text: "Contact", href: "/" },
];

const Navbar = (): ReactElement => {
  return (
    <div className="p-5 mx-auto bg-white z-auto">
      <div className="px-5 flex justify-between items-center">
        <div>
          <Image
            src="/images/logo-text.png"
            alt="truth memes logo"
            width={388}
            height={78}
          />
        </div>
        <div>
          <nav className="flex">
            <div className="flex space-x-2 items-center mr-6">
              <NavbarItem text="home" isActive={true} href="#" />
              <NavbarItem text="community" href="#" />
              <NavbarItem text="staking" href="#" />
              <NavbarItem text="memes" href="#" />
              <NavbarDropdown title="about" options={aboutOptions} />
            </div>
            <div className="mx-4">
              <ConnectButton />
            </div>
          </nav>
        </div>
      </div>
    </div>
  );
};

export default Navbar;
