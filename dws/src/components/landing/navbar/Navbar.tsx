import Image from "next/image";
import { ReactElement } from "react";
import NavbarItem from "./NavbarItem";

const Navbar = (): ReactElement => {
  return (
    <div className="p-5 mx-auto bg-white">
      <div className="px-5 flex justify-between items-center">
        <div>
          <Image
            src="/images/logo-text.png"
            alt="truth memes logo"
            width={388}
            height={78}
          />
        </div>
        <nav className="space-x-3 flex items-center">
          <NavbarItem text="home" isActive={true} href="#" />
          <NavbarItem text="community" href="#" />
          <NavbarItem text="staking" href="#" />
          <NavbarItem text="memes" href="#" />
          <NavbarItem text="about" href="#" />
        </nav>
      </div>
    </div>
  );
};

export default Navbar;
