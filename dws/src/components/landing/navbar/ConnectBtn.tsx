"use client";

import { Nullable } from "@/utils";
import { connectWallet } from "@/utils/metamask/wallet";
import { useState } from "react";
import toast from "react-hot-toast";

const ConnectButton = () => {
  const [account, setAccount] = useState<Nullable<string>>(null);

  const handleWalletConnect = async (): Promise<void> => {
    try {
      const userWallet = await connectWallet();
      if (userWallet == null) {
        return;
      }

      setAccount(userWallet);
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
      className="px-6 py-2 bg-cred text-white rounded-full border-4 border-black text-2xl tracking-wide uppercase"
      onClick={handleWalletConnect}
      disabled={account != null}
    >
      Connect
    </button>
  );
};

export default ConnectButton;
