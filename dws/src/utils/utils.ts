export const getExponentialWaitTime = (
  initialTime: number,
  time: number
): number => {
  return initialTime * Math.pow(2, time);
};

export const sleep = async (ms: number): Promise<void> => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};

/**
 *
 * @param date Date
 * @returns Date string in format 2023-10-22 08:08:38+02:00
 */
export const getRFC3339String = (date: Date): string => {
  const timezone = date.getTimezoneOffset();
  const timezoneHour = Math.abs(Math.floor(timezone / 60));

  const datePart = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
  const timePart = `${date.getHours()}:${date.getMinutes()}:${
    date.getSeconds() < 10 ? "0" : ""
  }${date.getSeconds()}`;

  const result = `${datePart} ${timePart}${
    timezone > 0 ? "-" : "+"
  }${timezoneHour}:00`;

  return result;
};
