import { getWalletShorthand } from "@/utils/metamask/wallet";
import { ReactElement, memo, useCallback } from "react";

interface WalletOptionProps {
  address: string;
  name: string;
  onSelect: (selectedWallet: string) => void;
}

const WalletOption = ({
  address,
  name,
  onSelect,
}: WalletOptionProps): ReactElement<WalletOptionProps> => {
  const handleSelect = useCallback(() => {
    onSelect(address);
  }, [address, onSelect]);

  return (
    <div
      className="flex flex-col border-2 border-black items-center justify-center px-4 py-2 rounded-lg box-border cursor-pointer hover:bg-gray-100"
      onClick={handleSelect}
    >
      <div className="text-lg pl-2">{name}</div>
      <div>{getWalletShorthand(address)}</div>
    </div>
  );
};

export default memo(WalletOption);
