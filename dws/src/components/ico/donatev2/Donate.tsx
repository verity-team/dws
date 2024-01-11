"use client";

import Image from "next/image";
import DonateForm from "./form/DonateForm";
import { useDonationData } from "@/api/dws/donation/donation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useToggle } from "@/hooks/utils/useToggle";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  LinearProgress,
} from "@mui/material";
import UserDonationStat from "../stats/user/UserDonationStat";

const Donate = () => {
  const {
    tokenPrices,
    donationStat: { tokens, total },
  } = useDonationData();

  const {
    isOpen: isHistoryOpen,
    open: openHistory,
    close: closeHistory,
  } = useToggle();

  const truthTokenPrice = useMemo(() => {
    if (!tokenPrices || tokenPrices.length === 0) {
      return "N/A";
    }

    const foundToken = tokenPrices.find((token) => token.asset === "truth");
    if (!foundToken) {
      return "N/A";
    }

    return foundToken.price;
  }, [tokenPrices]);

  const handleViewDonateHistory = useCallback(() => {
    openHistory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const targetSale = useMemo(() => {
    let value = Number(process.env.NEXT_PUBLIC_TARGET_SALE);
    if (isNaN(value)) {
      value = 10_500_000;
    }
    return value;
  }, []);

  const salePercent = useMemo(() => {
    let currentSale = Number(total);
    if (isNaN(currentSale)) {
      return 10;
    }

    const percent = (currentSale / targetSale) * 100;
    if (percent < 10) {
      return 10;
    }

    return percent;
  }, [total, targetSale]);

  return (
    <>
      <div className="flex flex-col transition-all">
        {/* Donate section */}
        <div className="max-w-2xl border-2 border-black rounded-2xl">
          <div className="w-full border-b-2 border-black bg-cred p-8 rounded-t-xl relative h-32">
            <Image
              src="/dws-images/givememoney.jpeg"
              alt="shut up and take my money"
              fill
              className="rounded-t-xl object-cover"
            />
          </div>

          <div className="border-b-2 border-black">
            <div className="mt-4 px-4">
              <div className="text-xl ml-1">
                <span className="text-cred">
                  {Number(tokens).toLocaleString()} $TRUTH
                </span>{" "}
                SOLD!
              </div>

              <LinearProgress
                variant="determinate"
                value={salePercent}
                color="warning"
                className="rounded-full h-6 border-2 border-black bg-white mt-1"
              />
              <div className="text-end">
                ${Number(total).toLocaleString()} / $
                {targetSale.toLocaleString()}
              </div>
            </div>

            <div className="flex items-center justify-between px-4 my-2">
              <div>Current price: ${truthTokenPrice} USD</div>
              <div
                className="text-blue-500 hover:text-blue-700 hover:underline cursor-pointer"
                onClick={handleViewDonateHistory}
              >
                History
              </div>
            </div>
          </div>

          <div className="bg-cblue py-2 rounded-b-2xl">
            <DonateForm
              tokenPrices={tokenPrices}
              rewardPrice={Number(truthTokenPrice)}
            />
          </div>
        </div>
      </div>
      <Dialog
        open={isHistoryOpen}
        onClose={closeHistory}
        fullWidth={true}
        maxWidth="sm"
        className="rounded-lg"
      >
        <DialogContent className="font-sans w-full">
          <UserDonationStat />
        </DialogContent>
      </Dialog>
    </>
  );
};

export default Donate;
