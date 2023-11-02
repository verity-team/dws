import Image from "next/image";
import { ReactElement } from "react";

interface ConnectStatusProps {
  walletLogo: string;
}

const ConnectStatus = ({
  walletLogo,
}: ConnectStatusProps): ReactElement<ConnectStatusProps> => {
  return (
    <div className="flex items-center justify-start border-2 border-black rounded-lg px-4 py-2">
      <div className="flex items-center justify-center">
        <Image
          src="/images/logo.png"
          alt="eye of truth"
          width={32}
          height={32}
        />
        -
        <Image src={walletLogo} alt="wallet logo" width={32} height={32} />
      </div>
      <div className="pl-4">
        <div className="text-xl">Connecting...</div>
        <div>
          Make sure to select all accounts that you want to grant access to
        </div>
      </div>
    </div>
  );
};

export default ConnectStatus;
