"use client";

import { ClientWallet, WalletUtils } from "@/components/ClientRoot";
import { Maybe } from "@/utils";
import { createSiweMesage } from "@/utils/siwe";
import { useContext, useEffect, useMemo, useState } from "react";

const SignInBtn = () => {
  const { connect, requestWalletSignature } = useContext(WalletUtils);
  const walletAddress = useContext(ClientWallet);

  const connected = useMemo(() => {
    if (walletAddress == null || walletAddress === "") {
      return false;
    }
    return true;
  }, [walletAddress]);

  useEffect(() => {
    if (walletAddress == null || walletAddress === "") {
      return;
    }

    const handleSignIn = async () => {
      const accessToken = localStorage.getItem("dws-at");
      if (accessToken != null && accessToken !== "") {
        // Verify access token with current wallet address
        const isAcessTokenValid = true;
        if (isAcessTokenValid) {
          return;
        }

        // If current wallet address and access token's wallet address mismatch
        localStorage.removeItem("dws-at");
      }

      // Request nonce and generate message
      const nonce = await Promise.resolve("somerandomnonce");

      // Request signature
      const message = createSiweMesage(walletAddress, {
        nonce,
        expirationTime: new Date().toISOString(),
        issuedAt: new Date().toISOString(),
      });
      await requestWalletSignature(message);
    };

    handleSignIn();
  }, [walletAddress]);

  return (
    <div>
      <button
        className="px-4 py-2 rounded-2xl bg-red-500 hover:bg-red-600 text-white cursor-pointer disabled:cursor-not-allowed"
        onClick={connect}
      >
        Sign in
      </button>
    </div>
  );
};

export default SignInBtn;
