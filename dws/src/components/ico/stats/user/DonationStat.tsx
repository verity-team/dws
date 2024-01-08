"use client";

import { Donation } from "@/api/dws/user/user.type";
import { getTimeElapsedString } from "@/utils/utils";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import Avatar from "boring-avatars";
import { ReactElement } from "react";

interface DonationStatProps {
  donation: Donation;
}

const DonationStat = ({
  donation,
}: DonationStatProps): ReactElement<DonationStatProps> => {
  const { amount, asset, price, status, tokens, ts, tx_hash, usd_amount } =
    donation;

  let donateTime = getTimeElapsedString(ts);

  return (
    <div>
      <div className="flex w-full items-center space-x-2">
        <Avatar size={40} name={tx_hash} variant="marble" />
        <div className="text-gray-700">{donateTime}</div>
      </div>
      <div className="mt-4 space-y-2">
        <div>
          Donated: {amount} <span className="text-blue-500">${asset}</span>
        </div>
        <div>
          Reward: {tokens} <span className="text-cred">$TRUTH</span>
        </div>
        <div>
          Status: <span className="uppercase">{status}</span>
        </div>
        <div>
          Transaction:{" "}
          <a
            href={`https://etherscan.io/tx/${tx_hash}`}
            target="_blank"
            className="text-blue-500 underline hover:text-blue-700 cursor-pointer"
          >
            {getWalletShorthand(tx_hash)}
          </a>
        </div>
      </div>
    </div>
  );
};

export default DonationStat;
