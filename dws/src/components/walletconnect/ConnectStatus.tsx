import Image from "next/image";
import { ReactElement, useCallback } from "react";
import { WalletConnectStatus } from "./ConnectModalv2";

interface ConnectStatusProps {
  walletLogo: string;
  status: WalletConnectStatus;
  onClick: () => void;
}

const ConnectStatus = ({
  walletLogo,
  status,
  onClick,
}: ConnectStatusProps): ReactElement<ConnectStatusProps> => {
  const handleRetry = useCallback(() => {
    if (status !== "rejected") {
      return;
    }

    onClick();
  }, [status, onClick]);

  return (
    <div
      // className="flex flex-col items-center justify-start border-2 border-black rounded-lg px-4 py-2 cursor-pointer"
      className="flex flex-col space-y-2 border-2 border-black rounded-lg p-4 md:flex-row"
      onClick={handleRetry}
    >
      <div className="flex items-center justify-center min-w-max">
        <Image
          src="/images/logo.png"
          alt="eye of truth"
          width={32}
          height={32}
        />
        -
        <Image src={walletLogo} alt="wallet logo" width={32} height={32} />
      </div>
      {status === "connecting" && (
        <div className="space-y-2 md:pl-4 md:space-y-0">
          <div className="text-xl text-center md:text-left">Connecting...</div>
          <div>
            Make sure to select all accounts that you want to grant access to
          </div>
        </div>
      )}
      {status === "pending" && (
        <div className="space-y-2 md:pl-4 md:space-y-0">
          <div className="text-xl text-center md:text-left">Connecting...</div>
          <div>
            MetaMask already has a pending connection request, please open the
            MetaMask app to login and connect
          </div>
        </div>
      )}
      {status === "rejected" && (
        <div className="space-y-2 md:pl-4 md:space-y-0">
          <div className="text-xl text-center md:text-left">
            Connection rejected!
          </div>
        </div>
      )}
    </div>
  );
};

export default ConnectStatus;
