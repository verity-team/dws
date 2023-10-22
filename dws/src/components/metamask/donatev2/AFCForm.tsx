"use client";

import { useUserDonationData } from "@/utils/api/clientAPI";
import { Modal, Box, Dialog, DialogTitle, DialogContent } from "@mui/material";
import { ReactElement, useEffect, useMemo, useState } from "react";

interface AFCFormProps {
  account: string;
}

const AFCForm = ({account}: AFCFormProps): ReactElement<AFCFormProps> => {
  const [isFormOpen, setFormOpen] = useState(false);

  const [userCode, setUserCode] = useState("");

  const {data, error, isLoading} = useUserDonationData(account)

  useEffect(() => {
    if(data == null) {
      return;
    }

    setUserCode(data.stats.)
  }, [data])

  const sharableLink = useMemo(() => {
    const baseUrl = window.location.href.split("?")[0];
    if (userCode == null || userCode.trim() === "") {
      return baseUrl;
    }

    return `${baseUrl}/afc=${userCode}`;
  }, [userCode]);

  const handleOpenForm = () => {
    setFormOpen(true);
  };

  const handleCloseForm = () => {
    setFormOpen(false);
  };

  const handleGenAffiliateCode = () => {};

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
      <Dialog open={isFormOpen} onClose={handleCloseForm}>
        <DialogTitle className="text-2xl">Share</DialogTitle>
        <DialogContent className="font-sans">
          <div>
            <div className="text-lg mb-2">Your affiliate code:</div>
            <button className="w-full px-4 py-2 border-2 border-black bg-blue-600 text-white rounded-lg">
              Generate affiliate code
            </button>
            <div className="text-base italic mt-2">
              Note: You will need to sign a message for verification purposes
            </div>
            <div className="mt-4">
              <div className="text-lg mb-2">Sharable link:</div>
              <input
                value={sharableLink}
                disabled
                className="p-4 border-2 border-black rounded-lg w-full"
              />
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default AFCForm;
