"use client";

import Image from "next/image";
import DonateForm from "./form/DonateForm";
import AFCForm from "./AFCForm";
import { useDonationData } from "@/api/dws/donation/donation";
import { useMemo } from "react";

const Donate = () => {
  const { tokenPrices } = useDonationData();

  const truthTokenPrice = useMemo(() => {
    if (!tokenPrices || tokenPrices.length === 0) {
      return "N/A";
    }

    const foundToken = tokenPrices.find((token) => token.asset === "truth");
    if (!foundToken) {
      return "N/A";
    }

    return foundToken.price;
  }, [tokenPrices]);

  return (
    <div className="flex flex-col transition-all">
      {/* Donate section */}
      <div className="max-w-2xl border-2 border-black rounded-2xl">
        <div className="w-full border-b-2 border-black bg-cred p-8 rounded-t-xl relative h-32">
          <Image
            src="/dws-images/givememoney.jpeg"
            alt="shut up and take my money"
            fill
            className="rounded-t-xl object-cover"
          />
        </div>

        <div className="border-b-2 border-black">
          <div className="flex items-center justify-between p-4">
            <div>Current price: ${truthTokenPrice} USD</div>
          </div>
        </div>

        <div className="bg-cblue py-2 rounded-b-2xl">
          <DonateForm
            tokenPrices={tokenPrices}
            rewardPrice={Number(truthTokenPrice)}
          />
        </div>
      </div>
    </div>
  );
};

export default Donate;
