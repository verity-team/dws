"use client";

import {
  requestNonce,
  verifyAccessToken,
  verifySignature,
} from "@/api/galactica/account/account";
import { ClientWallet, WalletUtils } from "@/components/ClientRoot";
import { createSiweMesage } from "@/utils/wallet/siwe";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { Dialog, DialogContent, DialogTitle } from "@mui/material";
import { useContext, useEffect, useMemo, useState } from "react";
import Image from "next/image";
import toast from "react-hot-toast";
import { sleep } from "@/utils/utils";

const SignInBtn = () => {
  const { connect, disconnect, requestWalletSignature } =
    useContext(WalletUtils);
  const walletAddress = useContext(ClientWallet);

  const [siweMessageOpen, setSiweMessageOpen] = useState(false);
  const [connected, setConnected] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (walletAddress == null || walletAddress === "") {
      return;
    }

    const handleSignIn = async (walletAddress: string) => {
      const accessToken = localStorage.getItem("dws-at");
      if (accessToken != null && accessToken !== "") {
        // Verify access token with current wallet address
        const isAcessTokenValid = await verifyAccessToken(walletAddress);

        if (isAcessTokenValid) {
          // No need to sign-in again
          handleConnectSuccess();
          return;
        }

        // If current wallet address and access token's wallet address mismatch
        localStorage.removeItem("dws-at");
      }

      await sleep(2000);
      setSiweMessageOpen(true);

      // Request nonce and generate message
      const nonce = await requestNonce();
      if (nonce == null) {
        // TODO: Maybe show some toast so user know the server is busy
        return;
      }

      // Request signature
      const message = createSiweMesage(walletAddress, nonce);

      let signature = "";
      try {
        signature = await requestWalletSignature(message);
      } catch (error) {
        setFailed(true);
        setTimeout(() => {
          handleCloseSiweMessage();
          disconnect();
        }, 2000);
      }

      const verifyResult = await verifySignature({ message, signature });
      if (verifyResult == null) {
        return;
      }

      // Store access token
      localStorage.setItem("dws-at", verifyResult.accessToken);
      handleConnectSuccess();
    };

    console.log(walletAddress);

    handleSignIn(walletAddress);
  }, [walletAddress]);

  const handleConnectSuccess = () => {
    handleCloseSiweMessage();
    setConnected(true);
    toast.success("Welcome to TruthMemes");
  };

  const handleCloseSiweMessage = () => {
    setSiweMessageOpen(false);
  };

  return (
    <div>
      {!connected && (
        <button
          className="px-4 py-2 rounded-2xl bg-red-500 hover:bg-red-600 text-white cursor-pointer disabled:cursor-not-allowed"
          onClick={connect}
        >
          Sign in
        </button>
      )}
      {connected && (
        <div className="text-2xl">
          Welcome {getWalletShorthand(walletAddress)}
        </div>
      )}
      <Dialog
        open={siweMessageOpen}
        onClose={handleCloseSiweMessage}
        fullWidth={true}
        maxWidth="sm"
        className="rounded-xl"
      >
        <DialogContent>
          <div className="flex flex-col items-center justify-center">
            <Image
              src="/images/logo.png"
              alt="eye of truth"
              width={64}
              height={64}
            />
            {failed ? (
              <div className="text-xl leading-10">
                Sign-in request declined. Please try again later
              </div>
            ) : (
              <div className="text-lg leading-10">
                Please approve the sign-in request using your wallet
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default SignInBtn;
