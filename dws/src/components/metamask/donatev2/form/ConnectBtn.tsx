import { Nullable } from "@/utils/types";
import { MouseEvent, ReactElement } from "react";

interface ConnectButtonProps {
  account: Nullable<string>;
  disabled: boolean;
  onConnect: (event: MouseEvent<HTMLButtonElement>) => void;
  onDonate: (event: MouseEvent<HTMLButtonElement>) => void;
}

const ConnectButton = ({
  account,
  disabled,
  onConnect,
  onDonate,
}: ConnectButtonProps): ReactElement<ConnectButtonProps> => {
  if (account != null) {
    return (
      <>
        <div className="font-sans text-sm text-center">
          Connected to {`${account.slice(0, 6)}...${account.slice(-4)}`}
          {/* <span
            className="text-blue-500 underline hover:text-blue-700 cursor-pointer ml-2"
            onClick={onChangeAccount}
          >
            Change account?
          </span> */}
        </div>

        <button
          className="w-full bg-cred border-2 border-black rounded-2xl py-2 disabled:opacity-60 disabled:cursor-not-allowed"
          onClick={onDonate}
          disabled={disabled}
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
      onClick={onConnect}
    >
      <div className="text-xl leading-loose tracking-wider text-gray-100">
        Connect Wallet
      </div>
    </button>
  );
};

export default ConnectButton;
