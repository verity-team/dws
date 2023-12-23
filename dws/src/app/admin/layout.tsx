"use client";

import ClientRoot from "@/components/ClientRoot";
import { ReactElement, ReactNode } from "react";

interface MemePageLayoutProps {
  children: ReactNode;
}

export default function AdminPageLayout({
  children,
}: MemePageLayoutProps): ReactElement<MemePageLayoutProps> {
  return (
    <ClientRoot>
      <div className="container mx-auto mt-12">{children}</div>
    </ClientRoot>
  );
}
