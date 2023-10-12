"use client";

import { useDonationData } from "@/utils/api/clientAPI";

const DonationStat = () => {
  const { tokenPrices, donationStat, receivingWallet, status, error } =
    useDonationData();

  return (
    <div className="mt-4">
      <div className="text-xl">Market prices</div>
      <div className="my-2 flex space-x-12">
        {tokenPrices.map((tokenPrice) => (
          <div key={tokenPrice.asset} className="my-2">
            <div>Asset: {tokenPrice.asset.toUpperCase()}</div>
            <div>Price: ${tokenPrice.price}</div>
            <div>At: {new Date(tokenPrice.ts).toLocaleString("en-US")}</div>
          </div>
        ))}
      </div>
      <div>Receving address: {receivingWallet}</div>
      <div className="my-2">
        <div>People have donated ${donationStat.total}</div>
        <div>Remaining reward: {donationStat.tokens} GMS</div>
      </div>
      <div className="my-2">Status: {status.toUpperCase()}</div>
    </div>
  );
};

export default DonationStat;
