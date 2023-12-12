"use client";

import { useAffiliateCode } from "@/hooks/useAffiliateCode";
import ClientRoot from "../ClientRoot";
import { ReactElement, ReactNode, createContext, useCallback } from "react";
import { connectWalletWithAffiliate } from "@/api/dws/affiliate/affiliate";
import { Maybe } from "@/utils";

interface LandingClientRootProps {
  children: ReactNode;
}

// Affiliate code
export const ClientAFC = createContext<Maybe<string>>(null);

const LandingClientRoot = ({
  children,
}: LandingClientRootProps): ReactElement<LandingClientRootProps> => {
  const affiliateCode = useAffiliateCode();

  const handleWalletConnect = useCallback(
    (address: string) => {
      let afc = "none";
      if (affiliateCode != null && affiliateCode !== "") {
        afc = affiliateCode;
      }
      connectWalletWithAffiliate({ address, code: afc });
    },
    [affiliateCode]
  );

  return (
    <ClientAFC.Provider value={affiliateCode}>
      <ClientRoot onWalletConnect={handleWalletConnect}>{children}</ClientRoot>
    </ClientAFC.Provider>
  );
};

export default LandingClientRoot;
