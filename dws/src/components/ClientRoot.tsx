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
import { Toaster } from "react-hot-toast";
import { connectWalletWithAffiliate } from "@/utils/api/client/affiliateAPI";
import ConnectModalV2 from "./wallet/ConnectModalv2";
import { AvailableToken, AvailableWallet } from "@/utils/token";
import { createWeb3Modal, defaultWagmiConfig } from "@web3modal/wagmi/react";
import { mainnet, sepolia } from "viem/chains";
import { WagmiConfig, useAccount, useSendTransaction } from "wagmi";
import { requestSignature } from "@/utils/metamask/sign";
import { donate } from "@/utils/metamask/donate";
import {
  prepareSendTransaction,
  sendTransaction,
  signMessage,
} from "@wagmi/core";
import { parseEther } from "viem";

interface ClientRootProps {
  children: ReactNode;
}

interface WalletUtils {
  connect: () => void;
  disconnect: () => void;
  requestTransaction: (
    amount: number,
    token: AvailableToken
  ) => Promise<string>;
  requestWalletSignature: (message: string) => Promise<string>;
}

// Affiliate code
export const ClientAFC = createContext<Nullable<string>>(null);
// Wallet address
export const ClientWallet = createContext<string>("");

// Functions to change wallet and disconnect wallet
export const WalletUtils = createContext<WalletUtils>({
  connect: () => {},
  disconnect: () => {},
  requestTransaction: () => Promise.resolve(""),
  requestWalletSignature: () => Promise.resolve(""),
});

const projectId = process.env.NEXT_PUBLIC_WC_PROJECT_ID ?? "";
const metadata = {
  name: "TruthMemes",
  description: "TruthMemes",
  url: "https://truthmemes.io/",
  icons: ["https://avatars.githubusercontent.com/u/37784886"],
};
const chains = [mainnet, sepolia];
const wagmiConfig = defaultWagmiConfig({ chains, projectId, metadata });

createWeb3Modal({
  wagmiConfig,
  projectId,
  chains,
  themeVariables: {
    "--w3m-z-index": 1400,
  },
  excludeWalletIds: [
    "c57ca95b47569778a828d19178114f4db188b89b763c899ba0be274e97267d96",
  ],
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

  const requestTransaction = useCallback(
    async (amount: number, token: AvailableToken) => {
      if (provider === "MetaMask") {
        const txHash = await donate(account, amount, token);
        if (txHash == null) {
          return "";
        }
        return txHash;
      }

      if (provider === "WalletConnect") {
        const receiver = process.env.NEXT_PUBLIC_DONATE_PUBKEY;
        if (!receiver) {
          console.warn("Receive wallet not set");
          return "";
        }
        const txConfig = await prepareSendTransaction({
          to: receiver,
          value: parseEther(amount.toString()),
        });
        const { hash } = await sendTransaction(txConfig);
        return hash;
      }

      return "";
    },
    [provider, account]
  );

  const requestWalletSignature = useCallback(
    async (message: string): Promise<string> => {
      if (provider === "MetaMask") {
        const signature = await requestSignature(account, message);
        if (signature == null) {
          return "";
        }
        return signature;
      }

      if (provider === "WalletConnect") {
        const signature = await signMessage({ message });
        return signature;
      }

      return "";
    },
    [provider, account]
  );

  const handleCloseConnectWalletForm = (): void => {
    setConnectWalletFormOpen(false);
  };

  return (
    <>
      <WagmiConfig config={wagmiConfig}>
        <WalletUtils.Provider
          value={{
            connect: connectWallet,
            disconnect: disconnectWallet,
            requestTransaction,
            requestWalletSignature: requestWalletSignature,
          }}
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
      </WagmiConfig>
    </>
  );
};

export default ClientRoot;
