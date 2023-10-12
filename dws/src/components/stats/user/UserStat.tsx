import { useUserDonationData } from "@/utils/api/clientAPI";
import { ReactElement } from "react";

interface UserStatProps {
  account: string;
}

const UserStat = ({ account }: UserStatProps): ReactElement<UserStatProps> => {
  // TODO: Replace this with account from props
  const { data: userDonationData, error } = useUserDonationData(
    "0xDEd1Fe6B3f61c8F1d874bb86F086D10FFc3F0154"
  );

  if (userDonationData == null) {
    return <div>User data unavailable</div>;
  }

  const { total, tokens, staked, reward, status } = userDonationData.stats;

  return (
    <div>
      <div className="text-xl">Donations</div>
      <div className="flex my-2 space-x-12">
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
        <div>Total reward: ${tokens} GMS</div>
        <div>Staking: {staked} GMS</div>
        <div>Staking reward: {reward} GMS</div>
        <div>Account status: {status.toUpperCase()}</div>
      </div>
    </div>
  );
};

export default UserStat;
