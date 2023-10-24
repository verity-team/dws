"use client";

import Image from "next/image";
import DonateForm from "./form/DonateForm";
import AFCForm from "./AFCForm";
import { Nullable } from "@/utils/types";
import { useContext, useState } from "react";
import toast from "react-hot-toast";
import { connectWallet } from "@/utils/metamask/wallet";
import { connectWalletWithAffiliate } from "@/utils/api/clientAPI";
import { ClientAFC } from "@/components/ClientRoot";

const Donate = () => {
  const affiliateCode = useContext(ClientAFC);

  const [account, setAccount] = useState<Nullable<string>>(
    "0xb938F65DfE303EdF96A511F1e7E3190f69036860"
  );

  const handleConnect = async (
    event: React.MouseEvent<HTMLButtonElement>
  ): Promise<void> => {
    event.preventDefault();

    try {
      const wallet = await connectWallet();
      if (wallet == null) {
        return;
      }
      setAccount(wallet);
      toast("Welcome to TruthMemes", { icon: "👋" });

      // Record wallet connection to server
      await connectWalletWithAffiliate({
        address: wallet,
        code: affiliateCode ?? "none",
      });
    } catch (err) {
      console.warn({ err });
    }
  };

  return (
    <div className="flex flex-col">
      {/* Donate section */}
      <div className="max-w-2xl border-2 border-black rounded-2xl">
        <div className="w-full border-b-2 border-black bg-cred p-8 rounded-t-xl relative h-32">
          <Image
            src="/images/givememoney.jpeg"
            alt="shut up and take my money"
            fill
            sizes="100vw 100vh"
            className="rounded-t-xl object-cover"
          />
        </div>

        <div className="border-b-2 border-black">
          <div className="flex items-center justify-between p-4">
            <div>Current price: 0</div>
            <div>Next stage price: 0</div>
          </div>
        </div>

        <div className="bg-cblue py-2 rounded-b-2xl">
          <DonateForm account={account} handleConnect={handleConnect} />
        </div>
      </div>

      {/* Share section */}
      <div className="mt-8">
        <AFCForm account={account ?? ""} />
      </div>
    </div>
  );
};

export default Donate;
