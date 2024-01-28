import { requestAccessTokenVerification } from "@/api/galactica/admin/admin";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AccountId {
  accessToken: string;
  getAccessToken: () => string;
  setAccessToken: (newAccessToken: string) => void;
}

const useAccountId = create<AccountId>()(
  persist(
    (set, get) => ({
      accessToken: "",
      getAccessToken: () => get().accessToken,
      setAccessToken: (newAccessToken) =>
        set(() => ({ accessToken: newAccessToken })),
      verifyAccessToken: async () => {
        const currentAccessToken = get().accessToken;
        if (!currentAccessToken) {
          return false;
        }

        const isValid =
          await requestAccessTokenVerification(currentAccessToken);
        return isValid;
      },
    }),
    {
      name: "dws-account-id",
    }
  )
);

export default useAccountId;
