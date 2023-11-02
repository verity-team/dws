import { Dialog, DialogContent, IconButton } from "@mui/material";
import Image from "next/image";
import { ReactElement, useCallback, useState } from "react";
import ConnectOption from "./ConnectOption";
import { AvailableWallet } from "@/utils/token";
import ConnectStatus from "./ConnectStatus";
import TextButton from "../common/TextButton";
import { connectWallet, requestAccounts } from "@/utils/metamask/wallet";
import WalletOption from "./WalletOption";
import CloseIcon from "@mui/icons-material/Close";

interface ConnectModalV2Props {
  isOpen: boolean;
  onClose: () => void;
}

/**
 * connecting - Default connect status
 *
 * pending - There are pending connect request(s),
 * and those must be finished before proceed with current request
 *
 * rejected - User refused to connect
 */
export type WalletConnectStatus = "connecting" | "pending" | "rejected";

const ConnectModalV2 = ({
  isOpen,
  onClose,
}: ConnectModalV2Props): ReactElement<ConnectModalV2Props> => {
  const [currentStep, setCurrentStep] = useState(0);
  const [currentProvider, setCurrentProvider] =
    useState<AvailableWallet>("MetaMask");

  const [accounts, setAccounts] = useState<string[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<string>("");

  const [connectStatus, setConnectStatus] =
    useState<WalletConnectStatus>("connecting");

  // Run procedure when closing connect wallet form
  const handleCloseModal = useCallback(() => {
    setCurrentStep(0);
    setConnectStatus("connecting");
    onClose();
  }, [onClose]);

  const handlePreviousStep = useCallback(() => {
    setCurrentStep((step) => (step - 1 < 0 ? 0 : step - 1));
    setConnectStatus("connecting");
  }, []);

  const handleConnectMetaMask = useCallback(async () => {
    // Make UI switch to connecting screen
    setCurrentStep(1);
    setCurrentProvider("MetaMask");

    let requestedAccounts = [];
    try {
      requestedAccounts = await requestAccounts();
    } catch (error: any) {
      console.log(error);
      if (error.code === 4001) {
        setConnectStatus("rejected");
      }

      if (error.code === -32002) {
        setConnectStatus("pending");
      }
      return;
    }

    if (requestedAccounts.length <= 0) {
      // Declare connection as failed
      setConnectStatus("rejected");
      return;
    }

    setAccounts(requestedAccounts);

    if (requestedAccounts.length === 1) {
      // User connect only 1 account
      // => Use that account, show success message (step 4), then proceed to close the popup
      setTimeout(() => {
        setCurrentStep(3);
        handleCloseModal();
      }, 3000);
    } else {
      // User connect multiple accounts, proceed to step 3
      setCurrentStep(2);
    }
  }, []);

  const handleRetry = useCallback(async () => {
    setConnectStatus("connecting");

    if (currentProvider === "MetaMask") {
      await handleConnectMetaMask();
    }
  }, [currentProvider, handleConnectMetaMask]);

  const handleSelectWallet = useCallback((selectedAccount: string) => {
    setSelectedAccount(selectedAccount);
    setCurrentStep(3);
  }, []);

  return (
    <Dialog
      open={isOpen}
      onClose={handleCloseModal}
      fullWidth={true}
      maxWidth="md"
      className="rounded-lg"
    >
      <IconButton
        aria-label="close"
        onClick={handleCloseModal}
        sx={{
          position: "absolute",
          right: 8,
          top: 8,
          color: (theme) => theme.palette.grey[500],
        }}
      >
        <CloseIcon />
      </IconButton>
      <DialogContent className="font-sans w-full">
        <div className="grid grid-cols-12">
          <div className="col-span-4 border-r border-black">
            <div className="p-4">
              <Image
                src="/images/logo.png"
                alt="eye of truth"
                width={64}
                height={64}
              />

              <div hidden={currentStep !== 0}>
                <div className="text-2xl py-2">Connect your wallet</div>
                <div className="py-1">
                  Connecting your wallet is like &quot;logging in&quot; to Web3.
                </div>
                <div className="py-1">
                  Select your wallet from the option to get started
                </div>
              </div>

              <div hidden={currentStep !== 1}>
                <div className="text-2xl py-2">Approve Connection</div>
                <div className="py-1">
                  Please approve the connection in your wallet and authorize
                  access to continue
                </div>
              </div>

              <div hidden={currentStep !== 2}>
                <div className="text-2xl py-2">Select Account</div>
                <div className="py-1">
                  We see that you have connected multiple wallets
                </div>
                <div className="py-1">
                  Please choose one to connect with our site
                </div>
                <div className="py-1 italic">
                  *You can switch between connected wallets. No worries!
                </div>
              </div>
            </div>
          </div>

          {/* Here I use hidden instead of conditional rendering to better support user redo */}
          <div className="col-span-8">
            <div className="px-4" hidden={currentStep !== 0}>
              <div className="text-xl p-2">Available Wallets (3)</div>
              <div className="grid grid-cols-2 gap-4 p-2">
                <ConnectOption
                  icon="/icons/metamask.svg"
                  name="MetaMask"
                  onClick={handleConnectMetaMask}
                />
              </div>
            </div>

            <div className="px-4" hidden={currentStep !== 1}>
              <div className="flex flex-col items-stretch justify-between h-full">
                <ConnectStatus
                  walletLogo="/icons/metamask.svg"
                  status={connectStatus}
                  onClick={handleRetry}
                />
              </div>
              <div className="my-8 flex items-center justify-center h-full">
                <TextButton onClick={handlePreviousStep}>
                  Back to wallets
                </TextButton>
              </div>
            </div>

            <div className="px-4" hidden={currentStep !== 2}>
              <div className="text-xl p-2">Available Wallets (3)</div>
              <div className="grid grid-cols-2 gap-4 p-2">
                {accounts.map((address, i) => (
                  <WalletOption
                    key={address}
                    address={address}
                    name={`Wallet ${i + 1}`}
                    selected={address === address}
                    onSelect={handleSelectWallet}
                  />
                ))}
              </div>
              <div className="my-8 flex items-center justify-center h-full space-x-2">
                <div>
                  <TextButton onClick={handlePreviousStep}>
                    Back to wallets
                  </TextButton>
                </div>
              </div>
            </div>

            <div className="px-4" hidden={currentStep !== 3}></div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectModalV2;
