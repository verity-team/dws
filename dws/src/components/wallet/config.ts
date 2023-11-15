import { InjectedConnector } from "@wagmi/core";
import { walletConnectProvider, EIP6963Connector } from "@web3modal/wagmi";
import { configureChains, mainnet, sepolia, createConfig } from "wagmi";
import { CoinbaseWalletConnector } from "wagmi/connectors/coinbaseWallet";
import { WalletConnectConnector } from "wagmi/connectors/walletConnect";
import { publicProvider } from "wagmi/providers/public";

// WalletConnect Project ID
const projectId = process.env.NEXT_PUBLIC_WC_PROJECT_ID ?? "";

// Wagmi Config
const { chains, publicClient } = configureChains(
  [mainnet, sepolia],
  [walletConnectProvider({ projectId }), publicProvider()]
);

// TruthMemes Website Metadata
const metadata = {
  name: "TruthMemes",
  description: "TruthMemes",
  url: "https://truthmemes.io/",
  icons: ["https://avatars.githubusercontent.com/u/37784886"],
};

export const wagmiConfig = createConfig({
  autoConnect: false,
  connectors: [
    new WalletConnectConnector({
      chains,
      options: { projectId, showQrModal: false, metadata },
    }),
    new EIP6963Connector({ chains }),
    new InjectedConnector({ chains, options: { shimDisconnect: true } }),
    new CoinbaseWalletConnector({
      chains,
      options: { appName: metadata.name },
    }),
  ],
  publicClient,
});

export const web3ModalConfig = {
  wagmiConfig,
  projectId,
  chains,
  themeVariables: {
    "--w3m-z-index": 1400,
  },
  excludeWalletIds: [
    // MetaMask
    "c57ca95b47569778a828d19178114f4db188b89b763c899ba0be274e97267d96",
  ],
  featuredWalletIds: [
    // TrustWallet
    "4622a2b2d6af1c9844944291e5e7351a6aa24cd7b23099efac1b2fd875da31a0",
    // Safe
    "225affb176778569276e484e1b92637ad061b01e13a048b35a9d280c3b58970f",
    // Ledger Live
    "19177a98252e07ddfc9af2083ba8e07ef627cb6103467ffebb3f8f4205fd7927",
  ],
};
