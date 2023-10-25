"use client";

import { Nullable } from "@/utils/types";
import { useSearchParams } from "next/navigation";
import React, {
  ReactElement,
  ReactNode,
  createContext,
  useCallback,
  useEffect,
  useState,
} from "react";
import toast, { Toaster } from "react-hot-toast";
import ConnectModal from "./metamask/wallet/ConnectModal";
import { connectWalletWithAffiliate } from "@/utils/api/clientAPI";
import { getWalletShorthand } from "@/utils/metamask/wallet";

interface ClientRootProps {
  children: ReactNode;
}

interface WalletUtils {
  connect: () => Promise<void>;
  disconnect: () => void;
}

// Affiliate code
export const ClientAFC = createContext<Nullable<string>>(null);
// Wallet address
export const ClientWallet = createContext<string>("");

// Functions to change wallet and disconnect wallet
export const WalletUtils = createContext<WalletUtils>({
  connect: () => Promise.resolve(),
  disconnect: () => {},
});

// For importing provider and all kind of wrapper for client components
const ClientRoot = ({
  children,
}: ClientRootProps): ReactElement<ClientRootProps> => {
  // Get affiliate code from URL
  const searchParams = useSearchParams();
  const affiliateCode = searchParams.get("afc");

  const [accounts, setAccounts] = useState<string[]>([]);
  const [account, setAccount] = useState("");
  const [selectWalletOpen, setSelectWalletOpen] = useState(false);

  useEffect(() => {
    if (account === "") {
      return;
    }

    toast.success(`Connected to ${getWalletShorthand(account)}`);
    toast("Welcome to TruthMemes", { icon: "👋" });
    connectWalletWithAffiliate({
      address: account,
      code: affiliateCode ?? "none",
    }).catch(console.warn);
  }, [account]);

  const connectWallet = useCallback(async (): Promise<void> => {
    const ethereum = window.ethereum;
    if (ethereum == null) {
      toast.error("No Ethereum wallet installed");
      return;
    }

    try {
      const accounts = await ethereum.request({
        method: "eth_requestAccounts",
      });
      console.log(accounts);
      if (accounts == null || !Array.isArray(accounts)) {
        return;
      }

      setAccounts(accounts);
    } catch (err) {
      console.warn({ err });
      return;
    }

    setSelectWalletOpen(true);
  }, []);

  const disconnectWallet = useCallback((): void => {
    setAccount("");
  }, []);

  const handleClose = (): void => {
    setSelectWalletOpen(false);
    setAccounts([]);
  };

  const handleSelectAccount = (selected: string): void => {
    if (selected === "") {
      return;
    }

    setAccount(selected);
  };

  return (
    <>
      <WalletUtils.Provider
        value={{ connect: connectWallet, disconnect: disconnectWallet }}
      >
        <ClientWallet.Provider value={account}>
          <ClientAFC.Provider value={affiliateCode}>
            {children}
          </ClientAFC.Provider>
        </ClientWallet.Provider>
      </WalletUtils.Provider>

      <Toaster />
      <ConnectModal
        isOpen={selectWalletOpen}
        accounts={accounts}
        onClose={handleClose}
        onSelect={handleSelectAccount}
      />
    </>
  );
};

export default ClientRoot;
