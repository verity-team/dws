"use client";

import { Dialog, DialogContent, IconButton } from "@mui/material";
import Image from "next/image";
import { ReactElement, SetStateAction, useCallback, useState } from "react";
import ConnectOption from "./ConnectOption";
import { AvailableWallet } from "@/utils/token";
import ConnectStatus from "./ConnectStatus";
import TextButton from "../common/TextButton";
import { getWalletShorthand, requestAccounts } from "@/utils/metamask/wallet";
import WalletOption from "./WalletOption";
import CloseIcon from "@mui/icons-material/Close";
import { sleep } from "@/utils/utils";
import { useWeb3Modal } from "@web3modal/wagmi/react";

interface ConnectModalV2Props {
  isOpen: boolean;
  onClose: () => void;
  selectedAccount: string;
  setSelectedAccount: React.Dispatch<SetStateAction<string>>;
  selectedProvider: AvailableWallet;
  setSelectedProvider: React.Dispatch<SetStateAction<AvailableWallet>>;
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
  selectedAccount,
  setSelectedAccount,
  selectedProvider,
  setSelectedProvider,
  onClose,
}: ConnectModalV2Props): ReactElement<ConnectModalV2Props> => {
  const [currentStep, setCurrentStep] = useState(0);

  const [accounts, setAccounts] = useState<string[]>([]);

  const [connectStatus, setConnectStatus] =
    useState<WalletConnectStatus>("connecting");

  const { open } = useWeb3Modal();

  // Run procedure when closing connect wallet form
  const handleCloseModal = useCallback(() => {
    onClose();

    setCurrentStep(0);
    setConnectStatus("connecting");
  }, [onClose]);

  const handleFinalizeConnection = useCallback(async () => {
    // Show success screen
    setCurrentStep(3);

    // Close modal after some times so user can read the message
    await sleep(2000);
    handleCloseModal();
  }, [handleCloseModal]);

  const handleBackToWallet = useCallback(() => {
    setCurrentStep(0);
    setConnectStatus("connecting");
  }, []);

  const handleConnectMetaMask = useCallback(async () => {
    // Make UI switch to connecting screen
    setCurrentStep(1);
    setSelectedProvider("MetaMask");

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
      setSelectedAccount(requestedAccounts[0]);
      handleFinalizeConnection();
      return;
    } else {
      // User connect multiple accounts, proceed to step 3
      setCurrentStep(2);
    }
  }, [handleFinalizeConnection, setSelectedAccount, setSelectedProvider]);

  const handleConnectWC = useCallback(() => {
    open();
  }, [open]);

  const handleRetry = useCallback(async () => {
    setConnectStatus("connecting");

    if (selectedProvider === "MetaMask") {
      await handleConnectMetaMask();
    }
  }, [selectedProvider, handleConnectMetaMask]);

  const handleSelectWallet = useCallback(
    (selectedAccount: string) => {
      setSelectedAccount(selectedAccount);
      handleFinalizeConnection();
    },
    [handleFinalizeConnection, setSelectedAccount]
  );

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

              <div hidden={currentStep !== 3}>
                <div className="text-2xl py-2">
                  Welcome to
                  <br />
                  Truth Memes
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
                <ConnectOption
                  icon="/icons/walletconnect.svg"
                  name="WalletConnect"
                  onClick={handleConnectWC}
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
                <TextButton onClick={handleBackToWallet}>
                  Back to wallets
                </TextButton>
              </div>
            </div>

            <div className="px-4" hidden={currentStep !== 2}>
              <div className="text-xl p-2">
                Connected wallets ({accounts.length})
              </div>
              <div className="grid grid-cols-2 gap-4 p-2">
                {accounts.map((address, i) => (
                  <WalletOption
                    key={address}
                    address={address}
                    name={`Wallet ${i + 1}`}
                    onSelect={handleSelectWallet}
                  />
                ))}
              </div>
              <div className="my-8 flex items-center justify-center h-full space-x-2">
                <div>
                  <TextButton onClick={handleBackToWallet}>
                    Back to wallets
                  </TextButton>
                </div>
              </div>
            </div>

            <div className="px-4" hidden={currentStep !== 3}>
              <div className="text-xl p-2">Connected</div>
              <div className="flex flex-col items-center justify-start border-2 border-black rounded-lg px-4 py-2">
                <div>Connected to {getWalletShorthand(selectedAccount)}</div>
                <div className="italic">
                  This screen will close in a short while
                </div>
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectModalV2;
