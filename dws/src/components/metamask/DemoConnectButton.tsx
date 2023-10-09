"use client";

import { useSDK } from "@metamask/sdk-react";
import { ReactElement, useContext, useState } from "react";
import { ClientAFC } from "../ClientRoot";

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

  return (
    <div className="w-screen h-screen flex flex-col items-center justify-center">
      <button
        type="button"
        className="px-4 py-2 rounded-lg border-2 border-black text-xl"
        onClick={handleWalletConnect}
      >
        Connect with wallet
      </button>
      <div className="mt-4 flex flex-col items-center">
        {connected && (
          <>
            {chainId && `Connected chain: ${chainId}`}
            <p></p>
            {account && `Connected account: ${account}`}
          </>
        )}
      </div>
    </div>
  );
};

export default DemoConnectButton;
