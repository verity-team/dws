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
  const maxSteps = donations.length;

  const handleNextDonation = useCallback(() => {
    setActiveDonation((current) => current + 1);
  }, []);

  const handlePrevDonation = useCallback(() => {
    setActiveDonation((current) => current - 1);
  }, []);

  return (
    <div className="flex flex-col items-center w-full" ref={ref}>
      <h3 className="text-4xl tracking-wide inline-block break-words mt-2 md:text-5xl">
        <span className="text-cred">Thanks</span> for your support
      </h3>

      <div className="w-full mt-4 md:flex md:flex-col md:items-center">
        <h3 className="text-2xl">Your donations</h3>
        <Box
          sx={{
            width: "100%",
            maxWidth: 900,
            flexGrow: 1,
            marginTop: "0.25rem",
          }}
        >
          <div className="bg-white border-2 border-b-0 border-black rounded-t-lg p-4">
            <DonationStat donation={donations[activeDonation]} />
          </div>
          <MobileStepper
            variant="text"
            position="static"
            className="bg-white border-2 border-black rounded-lg"
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
        </Box>
      </div>
      <div className="w-full mt-4 flex flex-col items-center">
        <h3 className="text-2xl tracking-wide inline-block break-words mt-2 md:text-5xl space-x-1">
          You have a total of
        </h3>
        <div className="text-3xl space-x-2">
          <span className="text-3xl">{userStat.total}</span>
          <span className="text-cred">$TRUTHMEME </span>!
        </div>
      </div>
    </div>
  );
};

export default forwardRef(UserStat);
