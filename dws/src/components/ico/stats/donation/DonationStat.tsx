"use client";

import { useDonationData } from "@/api/dws/donation/donation";

const DonationStat = () => {
  const { tokenPrices, donationStat, receivingWallet, status, error } =
    useDonationData();

  return (
    <div className="mt-4">
      <div className="text-xl">Market prices</div>
      <div className="flex space-x-12">
        {tokenPrices.map((tokenPrice) => (
          <div key={tokenPrice.asset} className="my-2">
            <div>Asset: {tokenPrice.asset.toUpperCase()}</div>
            <div>Price: ${tokenPrice.price}</div>
            <div>
              Updated at: {new Date(tokenPrice.ts).toLocaleString("en-GB")}
            </div>
          </div>
        ))}
      </div>
      <div>Receving address: {receivingWallet}</div>
      <div className="my-2">
        <div>People have donated ${donationStat.total}</div>
        <div>Token sold: {donationStat.tokens}</div>
      </div>
      <div className="my-2">Status: {status.toUpperCase()}</div>
    </div>
  );
};

export default DonationStat;
