"use client";

import { ClientWallet, WalletUtils } from "@/components/ClientRoot";
import { useContext, useEffect, useMemo, useState } from "react";

const SignInBtn = () => {
  const { connect } = useContext(WalletUtils);
  const wallet = useContext(ClientWallet);

  const connected = useMemo(() => {
    if (wallet == null || wallet === "") {
      return false;
    }
    return true;
  }, [wallet]);

  const handleSignIn = () => {
    connect();
  };

  return (
    <div>
      <button
        className="px-4 py-2 rounded-2xl bg-red-500 hover:bg-red-600 text-white"
        onClick={handleSignIn}
      >
        Sign in
      </button>
    </div>
  );
};

export default SignInBtn;
