"use client";

import React, {
  ReactElement,
  ReactNode,
  createContext,
  useCallback,
  useEffect,
  useState,
} from "react";
import toast, { Toaster } from "react-hot-toast";
import ConnectModalV2 from "./walletconnect/ConnectModalv2";
import { AvailableToken, AvailableWallet } from "@/utils/wallet/token";
import { createWeb3Modal } from "@web3modal/wagmi/react";
import { WagmiConfig } from "wagmi";
import { requestSignature } from "@/utils/wallet/sign";
import { donate } from "@/utils/wallet/donate";
import { connect, disconnect, getAccount, signMessage } from "@wagmi/core";
import { EstimateGasExecutionError } from "viem";
import { wagmiConfig, web3ModalConfig } from "./walletconnect/config";
import {
  LAST_PROVIDER_KEY,
  LAST_WALLET_KEY,
  NOT_ENOUGH_ERR,
  NO_BALANCE_ERR,
} from "@/utils/const";
import { requestAccounts } from "@/utils/wallet/wallet";

interface ClientRootProps {
  children: ReactNode;
  onWalletConnect?: (address: string, provider: AvailableWallet) => void;
}

interface IWalletUtils {
  connect: () => void;
  disconnect: () => void;
  requestTransaction: (
    amount: number,
    token: AvailableToken
  ) => Promise<string>;
  requestWalletSignature: (message: string) => Promise<string>;
}

interface IWallet {
  wallet: string;
  setWallet: (wallet: string) => void;
}

// Wallet address
export const Wallet = createContext<IWallet>({
  wallet: "",
  setWallet: () => {},
});

// Functions to change wallet and disconnect wallet
export const WalletUtils = createContext<IWalletUtils>({
  connect: () => {},
  disconnect: () => {},
  requestTransaction: () => Promise.resolve(""),
  requestWalletSignature: () => Promise.resolve(""),
});

// Init Web3Modal
createWeb3Modal(web3ModalConfig);

// For importing provider and all kind of wrapper for client components
const ClientRoot = ({
  children,
  onWalletConnect,
}: ClientRootProps): ReactElement<ClientRootProps> => {
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

    ethereum.on("accountsChanged", (accounts: string[]) => {
      if (accounts.length === 0) {
        setAccount("");
        connectWallet();
      }

      if (accounts[0] !== account) {
        setAccount(accounts[0]);
      }
    });
  }, []);

  useEffect(() => {
    const tryReconnect = async (
      lastWallet: string,
      lastProvider: AvailableWallet
    ) => {
      if (lastProvider === "MetaMask") {
        const connectedAccounts = await requestAccounts();
        if (connectedAccounts.includes(lastWallet)) {
          setAccount(lastWallet);
          setProvider(lastProvider);
          return;
        }

        if (connectedAccounts.length === 0) {
          return;
        }

        setAccount(connectedAccounts[0]);
        setProvider("MetaMask");
        return;
      }

      // Provider = WalletConnect
      const account = getAccount();
      if (!account?.address) {
        return;
      }
      setAccount(account.address);
      setProvider("WalletConnect");
    };

    const lastWallet = localStorage.getItem(LAST_WALLET_KEY);
    if (!lastWallet) {
      return;
    }

    const lastProvider = localStorage.getItem(
      LAST_PROVIDER_KEY
    ) as AvailableWallet;
    if (!lastProvider) {
      return;
    }

    tryReconnect(lastWallet, lastProvider);
  }, []);

  useEffect(() => {
    if (account === "") {
      return;
    }

    onWalletConnect?.(account, provider);
  }, [account, provider]);

  const connectWallet = useCallback(() => {
    setConnectWalletFormOpen(true);
  }, []);

  const disconnectWallet = useCallback((): void => {
    setAccount("");
    disconnect();
  }, []);

  const requestTransaction = useCallback(
    async (amount: number, token: AvailableToken) => {
      try {
        const txHash = await donate(account, amount, token, provider);

        if (txHash === NO_BALANCE_ERR) {
          toast.error(`No ${token} balance. Please try again later`);
          return "";
        }

        if (txHash === NOT_ENOUGH_ERR) {
          toast.error(`Your wallet does not have enough ${token} to proceed`);
          return "";
        }

        if (txHash == null) {
          // Will be catch under
          throw new Error();
        }
        return txHash;
      } catch (error: any) {
        if (error instanceof EstimateGasExecutionError) {
          toast.error("Insufficient fund");
        }
        return "";
      }
    },
    [provider, account]
  );

  const requestWalletSignature = useCallback(
    async (message: string): Promise<string> => {
      console.log(account);

      if (provider === "MetaMask") {
        const signature = await requestSignature(account, message);
        if (signature == null) {
          return "";
        }
        return signature;
      }

      if (provider === "WalletConnect") {
        try {
          const signature = await signMessage({ message });
          return signature;
        } catch (error: any) {
          console.warn(error);
        }
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
          <Wallet.Provider value={{ wallet: account, setWallet: setAccount }}>
            {children}
          </Wallet.Provider>
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
