"use client";

import ClientRoot from "@/components/ClientRoot";
import { ReactElement, ReactNode } from "react";

interface MemePageLayoutProps {
  children: ReactNode;
}

export default function MemePageLayout({
  children,
}: MemePageLayoutProps): ReactElement<MemePageLayoutProps> {
  return (
    <ClientRoot>
      <div className="container mt-12 mx-auto">{children}</div>
    </ClientRoot>
  );
}
