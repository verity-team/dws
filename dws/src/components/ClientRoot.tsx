"use client";

import { MetaMaskProvider } from "@metamask/sdk-react";
import React, { ReactElement, ReactNode, useMemo } from "react";

interface ClientRootProps {
  children: ReactNode;
}

// For importing provider and all kind of wrapper for client components
const ClientRoot = ({
  children,
}: ClientRootProps): ReactElement<ClientRootProps> => {
  const metamaskSettings = useMemo(
    () => ({
      logging: {
        developerMode: true,
      },
      checkInstallationImmediately: true,
      dappMetadata: {
        // TODO: Change these later
        name: "DWS",
        url: "http://localhost:3000",
      },
    }),
    [],
  );

  return (
    <MetaMaskProvider debug={true} sdkOptions={metamaskSettings}>
      {children}
    </MetaMaskProvider>
  );
};

export default ClientRoot;
