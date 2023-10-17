"use client";

import { theme } from "@/utils/theme";
import { Nullable } from "@/utils/types";
import { ThemeProvider } from "@emotion/react";
import { MetaMaskProvider } from "@metamask/sdk-react";
import { useSearchParams } from "next/navigation";
import React, { ReactElement, ReactNode, createContext, useMemo } from "react";

interface ClientRootProps {
  children: ReactNode;
}

// Affiliate code
export const ClientAFC = createContext<Nullable<string>>(null);

// For importing provider and all kind of wrapper for client components
const ClientRoot = ({
  children,
}: ClientRootProps): ReactElement<ClientRootProps> => {
  const metamaskSettings = useMemo(
    () => ({
      logging: {
        developerMode: true,
      },
      checkInstallationImmediately: false,
      dappMetadata: {
        // TODO: Change these later
        name: "DWS",
        url: "http://localhost:3000",
      },
    }),
    []
  );

  const searchParams = useSearchParams();
  const affliateCode = searchParams.get("afc");

  return (
    <ThemeProvider theme={theme}>
      <MetaMaskProvider debug={true} sdkOptions={metamaskSettings}>
        <ClientAFC.Provider value={affliateCode}>{children}</ClientAFC.Provider>
      </MetaMaskProvider>
    </ThemeProvider>
  );
};

export default ClientRoot;
