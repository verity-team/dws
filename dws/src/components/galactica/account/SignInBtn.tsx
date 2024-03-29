"use client";

import {
  verifyAccessToken,
  verifySignature,
} from "@/api/galactica/account/account";
import { Wallet, WalletUtils } from "@/components/ClientRoot";
import { createSiweMesage } from "@/utils/wallet/siwe";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import { Dialog, DialogContent } from "@mui/material";
import {
  ReactElement,
  memo,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import Image from "next/image";
import toast from "react-hot-toast";
import { useNonce } from "@/hooks/galactica/account/useNonce";
import useAccountId from "@/hooks/store/useAccountId";

type SignInBtnVariant = "button" | "text-only";

interface SignInBtnProps {
  variant?: SignInBtnVariant;
}

const SignInBtn = ({
  variant = "button",
}: SignInBtnProps): ReactElement<SignInBtnProps> => {
  const { connect, disconnect, requestWalletSignature } =
    useContext(WalletUtils);

  const userWallet = useContext(Wallet);

  const { accessToken, setAccessToken } = useAccountId();

  const { getNonce } = useNonce();

  const [siweMessageOpen, setSiweMessageOpen] = useState(false);
  const [txPending, setTxPending] = useState(false);
  const [connected, setConnected] = useState(false);
  const [failed, setFailed] = useState(false);

  const handleConnectSuccess = () => {
    handleCloseSiweMessage();
    setConnected(true);
    toast.success("Welcome to TruthMemes");
  };

  const handleCloseSiweMessage = useCallback(() => {
    setSiweMessageOpen(false);
  }, []);

  useEffect(() => {
    // Try to sign in
    if (connected || !userWallet.wallet || !accessToken) {
      return;
    }

    const trySignIn = async (): Promise<void> => {
      let validWalletAddress = await verifyAccessToken(
        userWallet.wallet,
        accessToken
      );
      if (!validWalletAddress) {
        // To be safe, try to remove access token from the local storage
        return;
      }

      // Successfully signed in
      setConnected(true);
    };

    trySignIn();
  }, []);

  const handleSignIn = async () => {
    setFailed(false);
    if (connected || txPending) {
      return;
    }

    if (!userWallet.wallet) {
      connect();
      return;
    }

    const walletAddress = userWallet.wallet;

    const accessTokenValid = await verifyAccessToken(
      userWallet.wallet,
      accessToken
    );
    if (accessTokenValid) {
      handleConnectSuccess();
      return;
    }

    setSiweMessageOpen(true);

    // Request nonce and generate message
    const nonce = await getNonce();
    if (nonce == null) {
      toast.error("Cannot contact server");
      setSiweMessageOpen(false);
      return;
    }

    // Request signature
    const message = createSiweMesage(walletAddress, nonce);

    let signature = "";
    try {
      setTxPending(true);
      signature = await requestWalletSignature(message);
    } catch (error) {
      setFailed(true);
      disconnect();
      setTimeout(() => {
        handleCloseSiweMessage();
      }, 2000);
      return;
    } finally {
      setTxPending(false);
    }

    const verifyResult = await verifySignature({ message, signature });
    if (verifyResult == null) {
      return;
    }

    // Store access token
    setAccessToken(verifyResult.accessToken);
    handleConnectSuccess();
  };

  useEffect(() => {
    if (connected || !userWallet.wallet) {
      return;
    }

    handleSignIn();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userWallet.wallet]);

  return (
    <div>
      {!connected && (
        <>
          {variant === "button" ? (
            <button
              className="px-4 py-2 rounded-2xl bg-red-500 hover:bg-red-600 text-white cursor-pointer disabled:cursor-not-allowed"
              onClick={handleSignIn}
            >
              Sign in
            </button>
          ) : (
            <div
              className="underline text-blue-700 hover:text-blue-900 cursor-pointer disabled:cursor-not-allowed"
              onClick={handleSignIn}
            >
              Sign in
            </div>
          )}
        </>
      )}
      {connected && (
        <>
          <div className="text-2xl hidden md:block">
            Welcome {getWalletShorthand(userWallet.wallet)}
          </div>
          <div className="text-2xl md:hidden">
            {getWalletShorthand(userWallet.wallet)}
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
              src="/dws-images/logo.png"
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
