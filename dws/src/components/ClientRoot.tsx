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
import { getWalletShorthand, requestAccounts } from "@/utils/metamask/wallet";
import { connectWalletWithAffiliate } from "@/utils/api/client/affiliateAPI";
import ConnectModalV2 from "./wallet/ConnectModalv2";

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
  const [connectWalletFormOpen, setConnectWalletFormOpen] = useState(false);

  useEffect(() => {
    const ethereum = window?.ethereum;
    if (ethereum == null) {
      return;
    }

    ethereum.on("chainChanged", (chainId: string) => {
      if (chainId === "0x1") {
        return;
      }

      window.location.reload();
    });
  }, []);

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
    const result = await requestAccounts();
    if (result == null || result.length === 0) {
      return;
    }

    setAccounts(result);
    setConnectWalletFormOpen(true);
  }, []);

  const disconnectWallet = useCallback((): void => {
    setAccount("");
  }, []);

  const handleCloseConnectWalletForm = (): void => {
    setConnectWalletFormOpen(false);
  };

  const handleSelectAccount = (selected: string): void => {
    if (selected === "") {
      return;
    }

    if (selected === account) {
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
      {/* <ConnectModal
        isOpen={selectWalletOpen}
        account={account}
        accounts={accounts}
        onClose={handleClose}
        onSelect={handleSelectAccount}
      /> */}
      <ConnectModalV2
        isOpen={connectWalletFormOpen}
        onClose={handleCloseConnectWalletForm}
      />
    </>
  );
};

export default ClientRoot;
