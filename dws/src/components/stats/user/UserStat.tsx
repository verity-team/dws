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
      <div className="flex space-x-12 mt-3">
        <div>
          {userDonationData.donations.map(
            ({ asset, amount, tokens, tx_hash, status, ts }) => (
              <div key={tx_hash} className="mb-4">
                <div>
                  Donated: {amount} {asset.toUpperCase()}
                </div>
                <div>Donated at: {new Date(ts).toLocaleString("en-GB")}</div>
                <div>Reward: {tokens}</div>
                <div>Status: {status.toUpperCase()}</div>
              </div>
            )
          )}
        </div>
        <div>
          <div>Total donated: ${total}</div>
          <div>Total reward: {tokens} GMS</div>
          <div>Staking: {staked} GMS</div>
          <div>Staking reward: {reward} GMS</div>
          <div>Account status: {status}</div>
        </div>
      </div>
    </div>
  );
};

export default UserStat;
