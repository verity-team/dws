import { Dialog, DialogTitle, DialogContent } from "@mui/material";
import { ReactElement } from "react";

interface ConnectModalProps {
  accounts: string[];
  isOpen: boolean;
  onClose: () => void;

  // Handle when user make their choise, whether it's cancel or confirm
  // Pass account = "" when user cancel => Not connected
  // Pass account = "..." when user choose one wallet => Connected
  onSelect: (account: string) => void;
}

const ConnectModal = ({
  accounts,
  isOpen,
  onClose,
}: ConnectModalProps): ReactElement<ConnectModalProps> => {
  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      fullWidth={true}
      maxWidth="sm"
      className="rounded-lg"
    >
      <DialogTitle className="text-2xl">Connect with wallet</DialogTitle>
      <DialogContent className="font-sans w-full">
        <div className="text-lg font-sans">
          <div>We detected more than one wallets in MetaMask</div>
          <div>Please choose your wallet to continue</div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectModal;
