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
    async (address: string, provider: AvailableWallet) => {
      let afc = "none";
      if (affiliateCode != null && affiliateCode !== "") {
        afc = affiliateCode;
      }

      localStorage.setItem(LAST_WALLET_KEY, address);
      localStorage.setItem(LAST_PROVIDER_KEY, provider);

      if (!affiliateCodeConnected) {
        await connectWalletWithAffiliate({ address, code: afc });
        setAffiliateCodeConnected(true);
      }
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
