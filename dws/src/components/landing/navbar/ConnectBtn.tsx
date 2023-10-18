"use client";

import { Nullable } from "@/utils/types";
import { useSDK } from "@metamask/sdk-react";
import { useState } from "react";
import toast from "react-hot-toast";

const ConnectButton = () => {
  const { sdk } = useSDK();

  const [account, setAccount] = useState<Nullable<string>>(null);

  const handleWalletConnect = async (): Promise<void> => {
    try {
      if (sdk == null) {
        return;
      }
      const accounts = await sdk.connect();

      if (accounts == null || !Array.isArray(accounts)) {
        return;
      }
      setAccount(accounts[0]);
      toast("Welcome to TruthMemes", { icon: "👋" });
    } catch (err) {
      console.warn({ err });
    }
  };

  if (account != null) {
    return <div></div>;
  }

  return (
    <button
      type="button"
      className="px-9 py-2 bg-cred text-white rounded-full border-4 border-black text-2xl tracking-wide uppercase"
      onClick={handleWalletConnect}
      disabled={account != null}
    >
      Connect
    </button>
  );
};

export default ConnectButton;
