"use client";

import { Donation } from "@/api/dws/user/user.type";
import { ClientWallet } from "@/components/ClientRoot";
import { getTimeElapsedString } from "@/utils/utils";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import Avatar from "boring-avatars";
import { time } from "console";
import { ReactElement, useContext, useMemo } from "react";

interface DonationStatProps {
  donation: Donation;
}

const DonationStat = ({
  donation,
}: DonationStatProps): ReactElement<DonationStatProps> => {
  const account = useContext(ClientWallet);

  const { amount, asset, price, status, tokens, ts, tx_hash, usd_amount } =
    donation;

  const donateTime = useMemo(() => {
    const timeString = getTimeElapsedString(ts);
    if (timeString === "just now") {
      return timeString;
    }

    return timeString + " ago";
  }, []);

  return (
    <div>
      <div className="flex w-full items-center space-x-2">
        <Avatar size={40} name={account} variant="marble" />
        <div className="text-gray-700">{donateTime}</div>
      </div>
      <div className="mt-4 space-y-2">
        <div>
          Donated: {Number(amount).toLocaleString()}{" "}
          <span className="text-blue-500">${asset}</span>
        </div>
        <div>
          Reward: {Number(tokens).toLocaleString()}{" "}
          <span className="text-cred">$TRUTHMEME</span>
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
