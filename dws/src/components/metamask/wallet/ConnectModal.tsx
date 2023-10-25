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
  onSelect,
}: ConnectModalProps): ReactElement<ConnectModalProps> => {
  const [selected, setSelected] = useState("");

  const handleSelectChange = useCallback(
    (event: FormEvent<HTMLInputElement>) => {
      setSelected(event.currentTarget.value);
    },
    []
  );

  const handleSubmit = () => {
    if (selected !== "") {
      onSelect(selected);
    }
    onClose();
  };

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
          <div className="flex items-center justify-end space-x-4">
            <button
              className="px-4 py-2 rounded-lg border-2 border-black text-white bg-red-500 hover:bg-red-600"
              onClick={handleSubmit}
            >
              Cancel
            </button>
            <button
              className="px-4 py-2 rounded-lg border-2 border-black text-white bg-blue-500 hover:bg-blue-600 disabled:opacity-60 disabled:cursor-not-allowed"
              disabled={selected === ""}
              onClick={handleSubmit}
            >
              Continue
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ConnectModal;
