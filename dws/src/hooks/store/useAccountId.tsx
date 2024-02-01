import { verifyAccessToken } from "@/api/galactica/account/account";
import { requestAccessTokenVerification } from "@/api/galactica/admin/admin";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AccountId {
  accessToken: string;
  setAccessToken: (newAccessToken: string) => void;
}

const useAccountId = create<AccountId>()(
  persist(
    (set, get) => ({
      accessToken: "",
      setAccessToken: (newAccessToken) =>
        set(() => ({ accessToken: newAccessToken })),
    }),
    {
      name: "dws-account-id",
    }
  )
);

export default useAccountId;
