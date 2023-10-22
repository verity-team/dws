"use client";

import { getUserDonationData } from "@/utils/api/clientAPI";
import { requestSignature } from "@/utils/metamask/sign";
import { useSDK } from "@metamask/sdk-react";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  CircularProgress,
} from "@mui/material";
import { ReactElement, useEffect, useMemo, useState } from "react";

interface AFCFormProps {
  account: string;
}

const AFCForm = ({ account }: AFCFormProps): ReactElement<AFCFormProps> => {
  const [isFormOpen, setFormOpen] = useState(false);
  const [isLoading, setLoading] = useState(true);

  const [userCode, setUserCode] = useState("");

  const { sdk } = useSDK();

  // Try to get user generated code
  useEffect(() => {
    if (account == null) {
      setLoading(false);
      return;
    }

    const getUserCode = async () => {
      const userDonationData = await getUserDonationData(account);
      if (userDonationData == null) {
        return;
      }

      return userDonationData.stats.affliate_code;
    };

    getUserCode()
      .then((code) => {
        if (code == null) {
          return;
        }
        setUserCode(code);
      })
      .finally(() => setLoading(false));
  }, [account]);

  const sharableLink = useMemo(() => {
    if (typeof window === "undefined") {
      return "";
    }

    const baseUrl = window.location.href.split("?")[0];
    if (userCode == null || userCode.trim() === "") {
      return baseUrl;
    }

    return `${baseUrl}?afc=${userCode}`;
  }, [userCode]);

  const handleOpenForm = () => {
    setFormOpen(true);
  };

  const handleCloseForm = () => {
    setFormOpen(false);

    // Reset form loading, but do NOT re-fetch user code
    setLoading(false);
  };

  const handleGenAffiliateCode = async () => {
    setLoading(true);

    // Try to (re)connect when there are no connected accounts
    let currentAccount = account;
    if (currentAccount == null || currentAccount === "") {
      try {
        if (sdk == null) {
          return;
        }
        const accounts = await sdk.connect();

        if (accounts == null || !Array.isArray(accounts)) {
          return;
        }
        currentAccount = accounts[0];
      } catch (err) {
        console.warn({ err });
      }
    }

    const currentDate = new Date();

    // Timestamp in seconds
    const timestamp = currentDate.getTime() / 1000;
    const timezone = currentDate.getTimezoneOffset();
    const timezoneHour = Math.abs(Math.floor(timezone / 60));
    const timezoneMinute = Math.abs(timezone % 60);

    const messageDate = `${currentDate.toISOString().split(".")[0]}${
      timezone > 0 ? "-" : "+"
    }${timezoneHour}:${timezoneMinute < 10 ? "0" : ""}${timezoneMinute}`;

    const message = `get affiliate code, ${messageDate}`;
    const encodedMessage = `0x${Buffer.from(message, "utf-8").toString("hex")}`;

    const signature = await requestSignature(currentAccount, encodedMessage);
    console.log(signature);
  };

  return (
    <>
      <div className="text-center text-4xl">
        Hype?{" "}
        <span
          className="underline cursor-pointer hover:text-blue-700"
          onClick={handleOpenForm}
        >
          Share now!
        </span>
      </div>
      <Dialog
        open={isFormOpen}
        onClose={handleCloseForm}
        fullWidth={true}
        maxWidth="sm"
        className="rounded-lg"
      >
        <DialogTitle className="text-2xl">Share</DialogTitle>
        <DialogContent className="font-sans w-full">
          <div>
            {!!userCode ? (
              <>
                <div className="text-lg mb-2">Your affiliate code:</div>
                <input
                  value={userCode}
                  disabled
                  className="px-4 py-2 border-2 border-black rounded-lg w-full text-center"
                />
              </>
            ) : (
              <>
                {isLoading ? (
                  <div className="w-full flex items-center justify-center py-2 border-2 border-black bg-blue-600 rounded-lg">
                    <CircularProgress size={24} className="text-white" />
                  </div>
                ) : (
                  <button
                    className="w-full px-4 py-2 border-2 border-black bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                    onClick={handleGenAffiliateCode}
                  >
                    Generate affiliate code
                  </button>
                )}
                <div className="text-base italic mt-2">
                  Note: You will need to sign a message for verification
                  purposes
                </div>
              </>
            )}

            <div className="mt-4">
              <div className="text-lg mb-2">Sharable link:</div>
              <input
                value={sharableLink}
                disabled
                className="px-4 py-2 border-2 border-black rounded-lg w-full"
              />
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default AFCForm;
