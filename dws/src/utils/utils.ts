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

  const breakpoint = /[T\.]/;
  const [datePart, timePart] = date.toISOString().split(breakpoint);

  const result = `${datePart} ${timePart}+00:00`;

  return result;
};
