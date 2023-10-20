import { Nullable } from "@/utils/types";
import { MouseEvent, ReactElement } from "react";

interface ConnectButtonProps {
  account: Nullable<string>;
  handleConnect: (event: MouseEvent<HTMLButtonElement>) => void;
  handleDonate: (event: MouseEvent<HTMLButtonElement>) => void;
}

const ConnectButton = ({
  account,
  handleConnect,
  handleDonate,
}: ConnectButtonProps): ReactElement<ConnectButtonProps> => {
  if (account != null) {
    return (
      <>
        <div className="font-sans text-sm text-center">
          Connected to {account}
        </div>
        <button
          className="w-full bg-cred border-2 border-black rounded-2xl py-2"
          onClick={handleDonate}
        >
          <div className="text-xl leading-loose tracking-wider text-gray-100">
            Donate
          </div>
        </button>
      </>
    );
  }

  return (
    <button
      className="w-full bg-cred border-2 border-black rounded-2xl py-2"
      onClick={handleConnect}
    >
      <div className="text-xl leading-loose tracking-wider text-gray-100">
        Connect Wallet
      </div>
    </button>
  );
};

export default ConnectButton;
