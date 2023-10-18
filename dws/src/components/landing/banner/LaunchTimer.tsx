"use client";

import { ReactElement, useEffect, useState } from "react";

interface TimerSpanProps {
  text: string | number;
  highlight?: boolean;
}

const TimerSpan = ({
  text,
  highlight = false,
}: TimerSpanProps): ReactElement<TimerSpanProps> => {
  if (highlight) {
    return <span className="text-cred text-4xl italic shadow-sm">{text}</span>;
  }
  return <span className="text-cblack text-4xl italic shadow-sm">{text}</span>;
};

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

  const [days, hours, minutes, seconds] = (() => {
    const launchDate = new Date(remainingTime);
    const days = Math.floor(remainingTime / (24 * 60 * 60 * 1000));

    const seconds = launchDate.getSeconds();
    let displaySeconds = "";
    if (seconds < 10) {
      displaySeconds = "0" + seconds;
    } else {
      displaySeconds = "" + seconds;
    }

    return [
      days,
      launchDate.getHours(),
      launchDate.getMinutes(),
      displaySeconds,
    ];
  })();

  return (
    <div className="space-x-2 max-w-full break-all break-words">
      <TimerSpan text={days} highlight={true} />
      <TimerSpan text="days" />

      <TimerSpan text={hours} highlight={true} />
      <TimerSpan text="hours" />

      <TimerSpan text={minutes} highlight={true} />
      <TimerSpan text="minutes" />

      <TimerSpan text={seconds} highlight={true} />
      <TimerSpan text="seconds" />
    </div>
  );
};

export default LaunchTimer;
