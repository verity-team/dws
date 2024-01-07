import { requestNonce } from "@/api/galactica/account/account";
import { NonceInfo } from "@/api/galactica/account/account.type";
import { Maybe } from "@/utils";
import { useCallback, useState } from "react";

export const useNonce = () => {
  const [nonce, setNonce] = useState<Maybe<NonceInfo>>(null);

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
