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
import {
  LAST_PROVIDER_KEY,
  LAST_WALLET_KEY,
  MAX_AFFILIATE_CODE_LENGTH,
} from "@/utils/const";
import { getDelphiMessage } from "@/utils/utils";
import { AvailableWallet } from "@/utils/wallet/token";
import { CircularProgress } from "@mui/material";
import { useSearchParams } from "next/navigation";
import { useAffiliateCode } from "@/hooks/useAffiliateCode";
import toast from "react-hot-toast";

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

      // a code longer than delphi accepts can only come from a broken or
      // tampered link; delphi would reject it with a generic validation error,
      // so tell the user here instead of asking for a signature first. The
      // length is counted in code points, like the spec's maxLength.
      if (afc !== "none" && [...afc].length > MAX_AFFILIATE_CODE_LENGTH) {
        toast.error("Affiliate code not found, no referral was recorded");
        setAffiliateCodeConnected(true);
        return;
      }

      // the backend requires proof that the caller owns `address`; the message
      // is derived from the endpoint path, the address, the timestamp and the
      // affiliate code sent in the body (so the code cannot be swapped on a
      // replayed signature) and must match what the backend rebuilds exactly
      const now = new Date();
      const timestamp = Math.floor(now.getTime() / 1000);
      const message = `${getDelphiMessage(
        "wallet connection",
        address,
        now
      )}, ${afc}`;

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

      const result = await connectWalletWithAffiliate({
        address,
        code: afc,
        timestamp,
        signature,
      });
      if (result === "unknown_code") {
        toast.error("Affiliate code not found, no referral was recorded");
      } else if (result === "self_referral") {
        toast.error("You cannot use your own affiliate code");
      }
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
