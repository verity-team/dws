"use client";

import { Wallet } from "@/components/ClientRoot";
import { useContext, useEffect, useState } from "react";
import UserStat from "./UserStat";
import { DONATION_PAGE_SIZE, useUserDonationData } from "@/api/dws/user/user";
import { ReplaceAll } from "lucide-react";

const UserDonationStat = () => {
  const userWallet = useContext(Wallet);

  // the donation history is served one page at a time; the backend refuses to
  // materialize an unbounded number of records for a single request
  const [page, setPage] = useState(0);

  useEffect(() => {
    setPage(0);
  }, [userWallet.wallet]);

  const { data: userDonationData } = useUserDonationData(
    userWallet.wallet,
    page
  );

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
      page={page}
      pageSize={DONATION_PAGE_SIZE}
      // a full page means there may well be another one behind it
      hasNextPage={userDonationData.donations.length === DONATION_PAGE_SIZE}
      onPageChange={setPage}
    />
  );
};

export default UserDonationStat;
