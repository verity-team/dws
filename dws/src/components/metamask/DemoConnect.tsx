"use client";

import { useSDK } from "@metamask/sdk-react";
import { ReactElement, useState } from "react";
import TextButton from "../common/TextButton";
import Donate from "./donate/Donate";
import UserStat from "../stats/user/UserStat";
import DonationStat from "../stats/donation/DonationStat";

const DemoConnect = (): ReactElement => {
  const { sdk } = useSDK();

  const [account, setAccount] = useState<string>();

  const handleWalletConnect = async (): Promise<void> => {
    try {
      if (sdk == null) {
        return;
      }
      const accounts = await sdk.connect();

      if (accounts == null || !Array.isArray(accounts)) {
        return;
      }
      setAccount(accounts[0]);
    } catch (err) {
      console.warn({ err });
    }
  };

  return (
    <div className="m-16">
      <div className="flex items-center space-x-2">
        <TextButton onClick={handleWalletConnect} disabled={account != null}>
          Connect
        </TextButton>
        {account && (
          <div>
            Connected with <b>{account}</b>
          </div>
        )}
      </div>
      <div className="my-2">{account && <Donate account={account} />}</div>
      <div className="flex space-x-12">
        {account && (
          <UserStat account={"0x379738c60f658601Be79e267e79cC38cEA07c8f2"} />
        )}
        <DonationStat />
      </div>
    </div>
  );
};

export default DemoConnect;
