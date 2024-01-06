"use client";

import { ClientWallet } from "@/components/ClientRoot";
import { useContext } from "react";
import UserStat from "./UserStat";
import { useUserDonationData } from "@/api/dws/user/user";

const UserDonationStat = () => {
  const account = useContext(ClientWallet);

  const { data: userDonationData, error } = useUserDonationData(account);

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
