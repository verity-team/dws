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
import { getWalletShorthand } from "@/utils/metamask/wallet";
import { connectWalletWithAffiliate } from "@/utils/api/client/affiliateAPI";
import ConnectModalV2 from "./wallet/ConnectModalv2";
import { AvailableWallet } from "@/utils/token";

interface ClientRootProps {
  children: ReactNode;
}

interface WalletUtils {
  connect: () => void;
  disconnect: () => void;
}

// Affiliate code
export const ClientAFC = createContext<Nullable<string>>(null);
// Wallet address
export const ClientWallet = createContext<string>("");

// Functions to change wallet and disconnect wallet
export const WalletUtils = createContext<WalletUtils>({
  connect: () => {},
  disconnect: () => {},
});

// For importing provider and all kind of wrapper for client components
const ClientRoot = ({
  children,
}: ClientRootProps): ReactElement<ClientRootProps> => {
  // Get affiliate code from URL
  const searchParams = useSearchParams();
  const affiliateCode = searchParams.get("afc");

  const [account, setAccount] = useState("");
  const [provider, setProvider] = useState<AvailableWallet>("MetaMask");
  const [connectWalletFormOpen, setConnectWalletFormOpen] = useState(false);

  useEffect(() => {
    const ethereum = window?.ethereum;
    if (ethereum == null) {
      return;
    }

    ethereum.on("chainChanged", (chainId: string) => {
      if (chainId === process.env.NEXT_PUBLIC_TARGET_NETWORK_ID) {
        return;
      }

      window.location.reload();
    });
  }, []);

  useEffect(() => {
    if (account === "") {
      return;
    }

    connectWalletWithAffiliate({
      address: account,
      code: affiliateCode ?? "none",
    }).catch(console.warn);
  }, [account]);

  const connectWallet = useCallback(() => {
    setConnectWalletFormOpen(true);
  }, []);

  const disconnectWallet = useCallback((): void => {
    setAccount("");
  }, []);

  const handleConfirmWalletAndProvider = (
    walletAddress: string,
    provider: AvailableWallet
  ) => {
    setAccount(walletAddress);
    setProvider(provider);
  };

  const handleCloseConnectWalletForm = (): void => {
    setConnectWalletFormOpen(false);
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
      <ConnectModalV2
        isOpen={connectWalletFormOpen}
        onClose={handleCloseConnectWalletForm}
        selectedAccount={account}
        setSelectedAccount={setAccount}
        selectedProvider={provider}
        setSelectedProvider={setProvider}
      />
    </>
  );
};

export default ClientRoot;
