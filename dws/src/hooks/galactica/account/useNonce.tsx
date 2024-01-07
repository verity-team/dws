import { requestNonce } from "@/api/galactica/account/account";
import { NonceInfo } from "@/api/galactica/account/account.type";
import { Maybe } from "@/utils";
import { useCallback, useEffect, useState } from "react";

export const useNonce = () => {
  const [nonce, setNonce] = useState<Maybe<NonceInfo>>(null);

  useEffect(() => {
    if (nonce == null) {
      return;
    }

    // Invalidat nonce when nonce reached its expiration time
    const exp = new Date(nonce.expirationTime);
    const timeToExpMs = exp.getTime() - Date.now();
    const timerId = setTimeout(() => {
      setNonce(null);
    }, timeToExpMs);

    return () => {
      clearTimeout(timerId);
    };
  }, [nonce]);

  const getNonce = useCallback(async (): Promise<Maybe<NonceInfo>> => {
    console.log("Current nonce");
    if (nonce != null) {
      return nonce;
    }

    const serverNonce = await requestNonce();
    if (serverNonce == null) {
      return null;
    }

    setNonce(serverNonce);
    return serverNonce;
  }, [nonce]);

  return {
    nonce,
    getNonce,
  };
};
