import { useUserDonationData } from "@/utils/api/clientAPI";
import { ReactElement, useEffect } from "react";

interface UserStatProps {
  account: string;
}

const UserStat = ({ account }: UserStatProps): ReactElement<UserStatProps> => {
  // TODO: Replace this with account from props
  const { data: userDonationData, error } = useUserDonationData(
    "0x379738c60f658601Be79e267e79cC38cEA07c8f2"
  );

  if (userDonationData == null || error != null) {
    console.log(error?.info);
    return <div>User data not available</div>;
  }

  const { total, tokens, staked, reward, status } = userDonationData.stats;

  return (
    <div className="mt-4">
      <div className="text-xl">Donations</div>
      <div className="flex space-x-12">
        <div className="mt-2 space-y-2">
          {userDonationData.donations.map(({ asset, amount, token }) => (
            <div key={asset}>
              <div>Asset: {asset.toUpperCase()}</div>
              <div>Amount: {amount}</div>
              <div>Reward: {token}</div>
            </div>
          ))}
        </div>
        <div className="my-4">
          <div>Total donated: ${total}</div>
          <div>Total reward: {tokens} GMS</div>
          <div>Staking: {staked} GMS</div>
          <div>Staking reward: {reward} GMS</div>
          <div>Account status: {status.toUpperCase()}</div>
        </div>
      </div>
    </div>
  );
};

export default UserStat;
