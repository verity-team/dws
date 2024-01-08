"use client";

import { Wallet } from "@/components/ClientRoot";
import { useContext } from "react";
import UserStat from "./UserStat";
import { useUserDonationData } from "@/api/dws/user/user";
// import { UserDonationData } from "@/api/dws/user/user.type";

const UserDonationStat = () => {
  const account = useContext(Wallet);

  const { data: userDonationData, error } = useUserDonationData(account);

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
  if (account == null || userDonationData == null) {
    return <div></div>;
  }

  return (
    <div className="md:ml-12">
      <UserStat
        donations={userDonationData.donations}
        userStat={userDonationData.user_data}
      />
    </div>
  );
};

export default UserDonationStat;
