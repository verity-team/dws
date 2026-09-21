"use client";

import { Donation, UserStats } from "@/api/dws/user/user.type";
import { MobileStepper, Button, Box } from "@mui/material";
import KeyboardArrowLeft from "@mui/icons-material/KeyboardArrowLeft";
import KeyboardArrowRight from "@mui/icons-material/KeyboardArrowRight";
import {
  ForwardedRef,
  ReactElement,
  forwardRef,
  useCallback,
  useState,
} from "react";
import DonationStat from "./DonationStat";

interface UserStatProps {
  donations: Donation[];
  userStat: UserStats;
  // index of the page of the donation history `donations` holds
  page: number;
  // number of records a full page holds
  pageSize: number;
  // whether there may be a page behind the current one
  hasNextPage: boolean;
  onPageChange: (page: number) => void;
}

const UserStat = (
  {
    donations,
    userStat,
    page,
    pageSize,
    hasNextPage,
    onPageChange,
  }: UserStatProps,
  ref: ForwardedRef<HTMLDivElement>
): ReactElement<UserStatProps> => {
  // index of the displayed donation *within the current page*
  const [activeDonation, setActiveDonation] = useState(0);
  const maxSteps = donations?.length ?? 0;
  // a page that is still loading, or one that shrank, must not be indexed out
  // of bounds
  const activeIndex = Math.min(
    Math.max(activeDonation, 0),
    Math.max(maxSteps - 1, 0)
  );
  const hasNext = activeIndex < maxSteps - 1 || hasNextPage;
  const hasPrev = activeIndex > 0 || page > 0;

  const handleNextDonation = useCallback(() => {
    if (activeIndex < maxSteps - 1) {
      setActiveDonation(activeIndex + 1);
      return;
    }
    if (hasNextPage) {
      // step over into the first record of the following page
      setActiveDonation(0);
      onPageChange(page + 1);
    }
  }, [activeIndex, maxSteps, hasNextPage, onPageChange, page]);

  const handlePrevDonation = useCallback(() => {
    if (activeIndex > 0) {
      setActiveDonation(activeIndex - 1);
      return;
    }
    if (page > 0) {
      // every page before the current one is full, so the last record of the
      // previous page is at pageSize - 1
      setActiveDonation(pageSize - 1);
      onPageChange(page - 1);
    }
  }, [activeIndex, page, pageSize, onPageChange]);

  // an empty array reaches the same conclusion as a missing one: there is no
  // donation to render, and `donations[activeIndex]` below would be undefined
  if (donations == null || donations.length === 0) {
    return <div></div>;
  }

  return (
    <div className="w-full font-changa" ref={ref}>
      <h3 className="text-xl">
        <span className="font-semibold">Total balance:</span> {userStat.tokens}{" "}
        <span className="text-cred">$TRUTH</span>
      </h3>
      <div className="mt-4">
        <h3 className="text-xl font-semibold">History</h3>
        <div className="bg-white rounded-t-lg mt-4">
          <DonationStat donation={donations[activeIndex]} />
        </div>
        <div className="mt-4">
          <MobileStepper
            variant="text"
            position="static"
            // the total is only known up to the page that was fetched; a full
            // page implies at least one more record behind it
            steps={page * pageSize + maxSteps + (hasNextPage ? 1 : 0)}
            activeStep={page * pageSize + activeIndex}
            nextButton={
              <Button
                size="small"
                onClick={handleNextDonation}
                disabled={!hasNext}
                className="text-black disabled:!text-gray-400 font-changa text-lg"
              >
                Next
                <KeyboardArrowRight />
              </Button>
            }
            backButton={
              <Button
                size="small"
                onClick={handlePrevDonation}
                disabled={!hasPrev}
                className="text-black disabled:!text-gray-400 font-changa text-lg"
              >
                <KeyboardArrowLeft />
                Back
              </Button>
            }
          />
        </div>
      </div>
    </div>
  );
};

export default forwardRef(UserStat);
