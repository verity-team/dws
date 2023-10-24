"use client";

import { Nullable } from "@/utils/types";
import { MetaMaskProvider } from "@metamask/sdk-react";
import { useSearchParams } from "next/navigation";
import React, { ReactElement, ReactNode, createContext, useMemo } from "react";
import { Toaster } from "react-hot-toast";

interface ClientRootProps {
  children: ReactNode;
}

// Affiliate code
export const ClientAFC = createContext<Nullable<string>>(null);

const metamaskSettings = {
  dappMetadata: {
    // TODO: Change these later
    name: "DWS",
    url: "http://localhost:3000",
  },
};

// For importing provider and all kind of wrapper for client components
const ClientRoot = ({
  children,
}: ClientRootProps): ReactElement<ClientRootProps> => {
  const searchParams = useSearchParams();
  const affiliateCode = searchParams.get("afc");

  return (
    <>
      <ClientAFC.Provider value={affiliateCode}>{children}</ClientAFC.Provider>
      <Toaster />
    </>
  );
};

export default ClientRoot;
