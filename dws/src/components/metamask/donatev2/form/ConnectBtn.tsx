import { WalletUtils } from "@/components/ClientRoot";
import { getWalletShorthand } from "@/utils/metamask/wallet";
import { CircularProgress } from "@mui/material";
import { MouseEvent, ReactElement, memo, useContext } from "react";

interface ConnectButtonProps {
  account: string;
  disabled: boolean;
  loading: boolean;
  onDonate: (event: MouseEvent<HTMLButtonElement>) => void;
}

const ConnectButton = ({
  account,
  disabled,
  loading,
  onDonate,
}: ConnectButtonProps): ReactElement<ConnectButtonProps> => {
  const { connect, disconnect } = useContext(WalletUtils);

  const handleConnect = async (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    await connect();
  };

  const handleChangeAccount = async () => {
    await connect();
  };

  if (account) {
    return (
      <>
        <div className="font-sans text-sm text-center">
          Connected to {getWalletShorthand(account)}
          <span
            className="text-blue-500 underline hover:text-blue-700 cursor-pointer ml-2"
            onClick={handleChangeAccount}
          >
            Change account?
          </span>
        </div>

        <button
          className="w-full bg-cred border-2 border-black rounded-2xl py-2 disabled:opacity-60 disabled:cursor-not-allowed"
          onClick={onDonate}
          disabled={disabled || loading}
        >
          {loading ? (
            <CircularProgress size={24} />
          ) : (
            <div className="text-xl leading-loose tracking-wider text-gray-100">
              Donate
            </div>
          )}
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

export default memo(ConnectButton);
