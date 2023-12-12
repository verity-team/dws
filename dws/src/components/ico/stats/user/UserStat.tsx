import { useUserDonationData } from "@/api/dws/donation/donation";
import { ReactElement } from "react";

interface UserStatProps {
  account: string;
}

const UserStat = ({ account }: UserStatProps): ReactElement<UserStatProps> => {
  // TODO: Replace this with account from props
  const { data: userDonationData, error } = useUserDonationData(account);

  if (userDonationData == null || error != null) {
    return <div className="mt-4 text-xl">User data not available</div>;
  }

  const { total, tokens, staked, reward, status } = userDonationData.user_data;

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
                <div>For: {tokens} token(s)</div>
                <div>Status: {status.toUpperCase()}</div>
              </div>
            )
          )}
        </div>
        <div>
          <div>Total donated: ${total}</div>
          <div>Token purchased: {tokens}</div>
          <div>Staking: {staked} token(s)</div>
          <div>Staking reward: {reward} token(s)</div>
          <div>Account status: {status}</div>
        </div>
      </div>
    </div>
  );
};

export default UserStat;
