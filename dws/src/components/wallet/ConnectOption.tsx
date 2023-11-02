import { ReactElement, ReactNode } from "react";
import WalletIcon from "@mui/icons-material/Wallet";
import { SvgIconTypeMap } from "@mui/material";
import { OverridableComponent } from "@mui/material/OverridableComponent";
import Image from "next/image";

interface ConnectOptionProps {
  icon?: string;
  name: string;
}

const ConnectOption = ({
  icon,
  name,
}: ConnectOptionProps): ReactElement<ConnectOptionProps> => {
  return (
    <div className="flex border-2 border-black items-center justify-center px-4 py-2 rounded-lg box-border cursor-pointer hover:bg-gray-100">
      <div>
        {icon ? (
          <Image src={icon} alt={`${name} icon`} width={32} height={32} />
        ) : (
          <WalletIcon fontSize="medium" />
        )}
      </div>
      <div className="text-lg pl-2">{name}</div>
    </div>
  );
};

export default ConnectOption;
