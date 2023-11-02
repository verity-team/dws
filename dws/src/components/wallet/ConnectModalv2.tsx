import { Dialog, DialogContent } from "@mui/material";
import Image from "next/image";
import { ReactElement } from "react";
import ConnectOption from "./ConnectOption";

interface ConnectModalV2Props {
  isOpen: boolean;
  onClose: () => void;
}

const ConnectModalV2 = ({
  isOpen,
  onClose,
}: ConnectModalV2Props): ReactElement<ConnectModalV2Props> => {
  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
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
            <div className="px-4">
              <div className="text-xl p-2">Available Wallets (3)</div>
              <div className="grid grid-cols-2 gap-4 p-2">
                <ConnectOption icon="/icons/metamask.svg" name="MetaMask" />
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectModalV2;
