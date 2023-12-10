"use client";

import { ClientWallet, WalletUtils } from "@/components/ClientRoot";
import { Maybe } from "@/utils";
import { useContext, useEffect, useMemo, useState } from "react";

// Try to get access token from local storage
const getAccessToken = (): Maybe<string> => {
  // TODO: Move this key to .env
  return localStorage.getItem("dws-at");
};

const SignInBtn = () => {
  const { connect } = useContext(WalletUtils);
  const wallet = useContext(ClientWallet);

  const connected = useMemo(() => {
    if (wallet == null || wallet === "") {
      return false;
    }
    return true;
  }, [wallet]);

  useEffect(() => {}, [wallet]);

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
