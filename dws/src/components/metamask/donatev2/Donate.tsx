"use client";

import Image from "next/image";
import DonateForm from "./form/DonateForm";
import AFCForm from "./AFCForm";
import { useSDK } from "@metamask/sdk-react";
import { Nullable } from "@/utils/types";
import { useState } from "react";
import toast from "react-hot-toast";
import { connectWallet } from "@/utils/metamask/wallet";

const Donate = () => {
  const { sdk } = useSDK();

  const [account, setAccount] = useState<Nullable<string>>(null);

  const handleConnect = async (
    event: React.MouseEvent<HTMLButtonElement>
  ): Promise<void> => {
    event.preventDefault();

    try {
      const wallet = await connectWallet(sdk);
      if (wallet == null) {
        return;
      }
      setAccount(wallet);
      toast("Welcome to TruthMemes", { icon: "👋" });
    } catch (err) {
      console.warn({ err });
    }
  };

  return (
    <div className="flex flex-col">
      <div className="max-w-2xl border-2 border-black rounded-2xl">
        <div className="w-full border-b-2 border-black bg-cred p-8 rounded-t-xl relative h-32">
          <Image
            src="/images/givememoney.jpeg"
            alt="shut up and take my money"
            fill={true}
            objectFit="cover"
            className="rounded-t-xl"
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
      <div className="mt-8">
        <AFCForm account={account ?? ""} />
      </div>
    </div>
  );
};

export default Donate;
