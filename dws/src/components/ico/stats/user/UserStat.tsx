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
}

const UserStat = (
  { donations, userStat }: UserStatProps,
  ref: ForwardedRef<HTMLDivElement>
): ReactElement<UserStatProps> => {
  const [activeDonation, setActiveDonation] = useState(0);
  const maxSteps = donations?.length ?? 0;

  const handleNextDonation = useCallback(() => {
    setActiveDonation((current) => current + 1);
  }, []);

  const handlePrevDonation = useCallback(() => {
    setActiveDonation((current) => current - 1);
  }, []);

  if (donations == null) {
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
          <DonationStat donation={donations[activeDonation]} />
        </div>
        <div className="mt-4">
          <MobileStepper
            variant="text"
            position="static"
            steps={maxSteps}
            activeStep={activeDonation}
            nextButton={
              <Button
                size="small"
                onClick={handleNextDonation}
                disabled={activeDonation === maxSteps - 1}
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
                disabled={activeDonation === 0}
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
