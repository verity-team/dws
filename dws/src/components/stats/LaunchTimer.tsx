"use client";

import { ReactElement, useEffect, useMemo, useState } from "react";

const LaunchTimer = (): ReactElement => {
  const [remainingTime, setRemainingTime] = useState<number>(
    Number(process.env.NEXT_PUBLIC_LAUNCH_TIME) - Date.now()
  );

  useEffect(() => {
    if (isNaN(remainingTime)) {
      console.log("Launch time not set");
      return;
    }
    const timerId = setInterval(() => {
      setRemainingTime((remainingTime) => remainingTime - 1000);
    }, 1000);

    return () => {
      clearInterval(timerId);
    };
  }, []);

  const displayTime = useMemo(() => {
    const launchDate = new Date(remainingTime);
    const days = Math.floor(remainingTime / (24 * 60 * 60 * 1000));
    return `${days} days, ${launchDate.getHours()} hours ${launchDate.getMinutes()} minutes ${launchDate.getSeconds()} seconds`;
  }, [remainingTime]);

  return <div>{displayTime}</div>;
};

export default LaunchTimer;
