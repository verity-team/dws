import { getWalletShorthand } from "@/utils/metamask/wallet";
import { Dialog, DialogTitle, DialogContent } from "@mui/material";
import { FormEvent, ReactElement, useCallback, useState } from "react";

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
  const [selected, setSelected] = useState("");

  const handleSelectChange = useCallback(
    (event: FormEvent<HTMLInputElement>) => {
      setSelected(event.currentTarget.value);
    },
    []
  );

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
          <div className="space-y-2 mt-4">
            {accounts.map((wallet, i) => (
              <div key={wallet} className="flex items-center space-x-4">
                <input
                  type="radio"
                  name="wallet"
                  id={`wallet ${i}`}
                  value={wallet}
                  checked={wallet === selected}
                  onChange={handleSelectChange}
                />
                <label htmlFor={`wallet ${i}`}>
                  <div>Wallet {i + 1}</div>
                  <div>{getWalletShorthand(wallet)}</div>
                </label>
              </div>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectModal;
