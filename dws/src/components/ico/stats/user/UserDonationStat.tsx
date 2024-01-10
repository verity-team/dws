"use client";

import { Wallet } from "@/components/ClientRoot";
import { useContext } from "react";
import UserStat from "./UserStat";
import { useUserDonationData } from "@/api/dws/user/user";
import { ReplaceAll } from "lucide-react";

const UserDonationStat = () => {
  const userWallet = useContext(Wallet);

  const { data: userDonationData } = useUserDonationData(userWallet.wallet);

  // const userDonationData: UserDonationData = {
  //   donations: [
  //     {
  //       amount: "1000",
  //       asset: "USDT",
  //       price: "1",
  //       status: "confirmed",
  //       ts: "2024-01-06T14:21:36.504Z",
  //       tokens: "20000",
  //       tx_hash:
  //         "0x558c8b86f75b0149ddca028a44a302eebe47c032e8ca8834f07aa6f1c5ae3fac",
  //       usd_amount: "1000",
  //     },
  //     {
  //       amount: "3",
  //       asset: "ETH",
  //       price: "1",
  //       status: "confirmed",
  //       ts: "2024-01-06T14:21:36.504Z",
  //       tokens: "20000",
  //       tx_hash:
  //         "0x558c8b86f75b0149ddca028a44a302eebe47c032e8ca8834f07aa6f1c5ae3fac",
  //       usd_amount: "1000",
  //     },
  //     {
  //       amount: "1000",
  //       asset: "USDT",
  //       price: "1",
  //       status: "confirmed",
  //       ts: "2024-01-06T14:21:36.504Z",
  //       tokens: "20000",
  //       tx_hash:
  //         "0x558c8b86f75b0149ddca028a44a302eebe47c032e8ca8834f07aa6f1c5ae3fac",
  //       usd_amount: "1000",
  //     },
  //   ],
  //   user_data: {
  //     total: "31415",
  //     tokens: "9880000",
  //     staked: "979999",
  //     reward: "979",
  //     status: "staking",
  //     ts: "2023-10-05T09:23:10+00:00",
  //     affiliate_code: "GezIkgeubrOkiOv4",
  //   },
  // };

  // User have not connected, or there are no data on this user
  // Simply skip user stats rendering
  if (userWallet == null || userDonationData?.donations == null) {
    return (
      <div className="w-full font-changa">
        <div className="flex flex-col items-center justify-center">
          <ReplaceAll size={64} color="#64748b" />
          <div className="text-lg mt-4">
            Your transaction history is currently empty
          </div>
          <div className="text-sm italic mt-4 font-roboto">
            * It might take a few minutes for your transaction to be recorded on
            the blockchain. We will try out best to deliver it as soon as
            possible
          </div>
        </div>
      </div>
    );
  }

  return (
    <UserStat
      donations={userDonationData.donations}
      userStat={userDonationData.user_data}
    />
  );
};

export default UserDonationStat;
