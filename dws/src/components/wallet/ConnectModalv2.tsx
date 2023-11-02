import { Dialog, DialogContent } from "@mui/material";
import Image from "next/image";
import { ReactElement, useCallback, useState } from "react";
import ConnectOption from "./ConnectOption";
import { AvailableWallet } from "@/utils/token";
import ConnectStatus from "./ConnectStatus";
import TextButton from "../common/TextButton";

interface ConnectModalV2Props {
  isOpen: boolean;
  onClose: () => void;
}

const ConnectModalV2 = ({
  isOpen,
  onClose,
}: ConnectModalV2Props): ReactElement<ConnectModalV2Props> => {
  const [currentStep, setCurrentStep] = useState(0);
  const [currentProvider, setCurrentProvider] =
    useState<AvailableWallet>("MetaMask");

  // Run procedure when closing connect wallet form
  const handleCloseModal = useCallback(() => {
    setCurrentStep(0);
    onClose();
  }, [onClose]);

  const handlePreviousStep = useCallback(() => {
    setCurrentStep((step) => (step - 1 < 0 ? 0 : step - 1));
  }, []);

  const handleConnectMetaMask = () => {
    // Make UI switch to connecting screen
    setCurrentStep(1);
    setCurrentProvider("MetaMask");

    console.log("Connecting to MetaMask");
  };

  return (
    <Dialog
      open={isOpen}
      onClose={handleCloseModal}
      fullWidth={true}
      maxWidth="md"
      className="rounded-lg"
    >
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
              <div className="text-2xl py-2">Connect your wallet</div>
              <div className="py-1">
                Connecting your wallet is like &quot;logging in&quot; to Web3.
              </div>
              <div className="py-1">
                Select your wallet from the option to get started
              </div>
            </div>
          </div>

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
                <ConnectStatus walletLogo="/icons/metamask.svg" />
              </div>
              <div className="my-8 flex items-center justify-center h-full">
                <TextButton onClick={handlePreviousStep}>
                  Back to wallets
                </TextButton>
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectModalV2;
