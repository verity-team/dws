"use client";

import ClientRoot from "../ClientRoot";
import {
  ReactElement,
  ReactNode,
  Suspense,
  createContext,
  useCallback,
  useState,
} from "react";
import { connectWalletWithAffiliate } from "@/api/dws/affiliate/affiliate";
import { Maybe } from "@/utils";
import { LAST_PROVIDER_KEY, LAST_WALLET_KEY } from "@/utils/const";
import { getRFC3339String } from "@/utils/utils";
import { AvailableWallet } from "@/utils/wallet/token";
import { CircularProgress } from "@mui/material";
import { useSearchParams } from "next/navigation";
import { useAffiliateCode } from "@/hooks/useAffiliateCode";

interface LandingClientRootProps {
  children: ReactNode;
}

// Affiliate code
export const ClientAFC = createContext<Maybe<string>>(null);

const LandingClientRoot = ({
  children,
}: LandingClientRootProps): ReactElement<LandingClientRootProps> => {
  const affiliateCode = useAffiliateCode();

  const [affiliateCodeConnected, setAffiliateCodeConnected] = useState(false);

  const handleWalletConnect = useCallback(
    async (
      address: string,
      provider: AvailableWallet,
      requestWalletSignature: (message: string) => Promise<string>
    ) => {
      let afc = "none";
      if (affiliateCode != null && affiliateCode !== "") {
        afc = affiliateCode;
      }

      localStorage.setItem(LAST_WALLET_KEY, address);
      localStorage.setItem(LAST_PROVIDER_KEY, provider);

      if (affiliateCodeConnected) {
        return;
      }

      // the backend requires proof that the caller owns `address`; the message
      // is derived from the endpoint path and must match it exactly
      const now = new Date();
      const timestamp = Math.floor(now.getTime() / 1000);
      const message = `wallet connection, ${getRFC3339String(now)}`;

      let signature = "";
      try {
        signature = await requestWalletSignature(message);
      } catch {
        // the user declined to sign; nothing to report here
        return;
      }

      if (!signature) {
        return;
      }

      await connectWalletWithAffiliate({
        address,
        code: afc,
        timestamp,
        signature,
      });
      setAffiliateCodeConnected(true);
    },
    [affiliateCode, affiliateCodeConnected]
  );

  return (
    <ClientAFC.Provider value={affiliateCode}>
      <ClientRoot onWalletConnect={handleWalletConnect}>{children}</ClientRoot>
    </ClientAFC.Provider>
  );
};

export default LandingClientRoot;
