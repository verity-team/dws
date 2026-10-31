import { Maybe } from "@/utils";
import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

// Former persist key; it held the access token in localStorage
const LEGACY_STORAGE_KEY = "dws-account-id";
const STORAGE_KEY = "dws-access-token";

interface AccountId {
  accessToken: string;
  setAccessToken: (newAccessToken: string) => void;
}

// Expiry (seconds since the epoch) of a JWT access token, or null when the
// token has no readable `exp` claim. The claim is not verified -- galactica
// does that on every request -- it only lets a stored token that has expired be
// dropped instead of reused.
const tokenExpiry = (token: string): Maybe<number> => {
  const parts = token.split(".");
  if (parts.length !== 3) {
    return null;
  }

  try {
    const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, "=");
    const { exp } = JSON.parse(atob(padded));
    return typeof exp === "number" ? exp : null;
  } catch {
    return null;
  }
};

const isExpired = (token: string): boolean => {
  const exp = tokenExpiry(token);
  return exp != null && exp * 1000 <= Date.now();
};

// The access token is kept in sessionStorage rather than localStorage: it
// survives reloads and in-tab navigation but is gone once the tab is closed, so
// an XSS can no longer pick it up from any later visit. An expired token is not
// restored.
const useAccountId = create<AccountId>()(
  persist(
    (set) => ({
      accessToken: "",
      setAccessToken: (newAccessToken) =>
        set(() => ({ accessToken: newAccessToken })),
    }),
    {
      name: STORAGE_KEY,
      storage: createJSONStorage(() => sessionStorage),
      partialize: ({ accessToken }) => ({ accessToken }),
      merge: (persisted, current) => {
        const token = (persisted as Partial<AccountId> | undefined)
          ?.accessToken;
        if (typeof token !== "string" || token === "" || isExpired(token)) {
          return { ...current, accessToken: "" };
        }
        return { ...current, accessToken: token };
      },
    }
  )
);

if (typeof window !== "undefined") {
  try {
    window.localStorage.removeItem(LEGACY_STORAGE_KEY);
  } catch (error) {
    console.warn("Cannot clear the stored access token", error);
  }
}

export default useAccountId;
