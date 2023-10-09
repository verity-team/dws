"use client";

import { useSDK } from "@metamask/sdk-react";
import { ReactElement, useMemo, useState } from "react";
import TextButton from "../common/TextButton";
import DonateForm from "./donate/DonateForm";

const DemoConnectButton = (): ReactElement => {
  const { sdk, connected, chainId } = useSDK();

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

  const userConnected: boolean = useMemo(
    () => account != null && connected,
    [account, connected]
  );

  return (
    <div className="p-16">
      <div className="flex items-center space-x-2">
        <TextButton onClick={handleWalletConnect}>
          Connect with wallet
        </TextButton>
        {userConnected && (
          <div>
            Connected with <b>{account}</b>
          </div>
        )}
      </div>
      {account != null && connected && <DonateForm account={account} />}
    </div>
  );
};

export default DemoConnectButton;
