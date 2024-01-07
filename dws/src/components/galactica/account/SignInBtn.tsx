"use client";

import {
  requestNonce,
  verifyAccessToken,
  verifySignature,
} from "@/api/galactica/account/account";
import { ClientWallet, WalletUtils } from "@/components/ClientRoot";
import { createSiweMesage } from "@/utils/wallet/siwe";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { Dialog, DialogContent } from "@mui/material";
import { memo, useContext, useEffect, useState } from "react";
import Image from "next/image";
import toast from "react-hot-toast";
import { DWS_AT_KEY } from "@/utils/const";
import { useNonce } from "@/hooks/galactica/account/useNonce";

const verifyUserAccessToken = async (wallet: string): Promise<boolean> => {
  const accessToken = localStorage.getItem(DWS_AT_KEY);

  if (accessToken == null || accessToken.trim() === "") {
    return false;
  }

  // Verify access token with current wallet address
  const isAcessTokenValid = await verifyAccessToken(wallet);

  if (isAcessTokenValid) {
    // No need to sign-in again
    return true;
  }

  // If current wallet address and access token's wallet address mismatch
  localStorage.removeItem(DWS_AT_KEY);
  return false;
};

const SignInBtn = () => {
  const { connect, disconnect, requestWalletSignature } =
    useContext(WalletUtils);
  const walletAddress = useContext(ClientWallet);

  const { getNonce } = useNonce();

  const [siweMessageOpen, setSiweMessageOpen] = useState(false);
  const [signaturePending, setSignaturePending] = useState(false);

  const [connected, setConnected] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (walletAddress == null || walletAddress === "" || signaturePending) {
      return;
    }

    const handleSignIn = async (walletAddress: string) => {
      setSignaturePending(true);

      const accessTokenValid = await verifyUserAccessToken(walletAddress);
      if (accessTokenValid) {
        handleConnectSuccess();
        return;
      }

      setTimeout(() => setSiweMessageOpen(true), 2000);

      // Request nonce and generate message
      const nonce = await getNonce();
      if (nonce == null) {
        // TODO: Maybe show some toast so user know the server is busy
        return;
      }

      console.log(nonce);

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
        return;
      }

      const verifyResult = await verifySignature({ message, signature });
      if (verifyResult == null) {
        return;
      }

      // Store access token
      localStorage.setItem(DWS_AT_KEY, verifyResult.accessToken);
      handleConnectSuccess();
    };

    handleSignIn(walletAddress).finally(() => setSignaturePending(false));
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
        <>
          <div className="text-2xl hidden md:block">
            Welcome {getWalletShorthand(walletAddress)}
          </div>
          <div className="text-2xl md:hidden">
            {getWalletShorthand(walletAddress)}
          </div>
        </>
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

export default memo(SignInBtn);
