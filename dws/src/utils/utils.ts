import dayjs from "dayjs";

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
  const breakpoint = /[T\.]/;
  const [datePart, timePart] = date.toISOString().split(breakpoint);
  const result = `${datePart} ${timePart}+00:00`;
  return result;
};

/**
 * Build the message the delphi backend expects the caller to sign.
 *
 * It is made up of the words of the endpoint path, the lower cased address the
 * caller claims to own and the UTC timestamp. The address is part of the
 * message so that the signature is bound to the account it is used for, and
 * the backend reconstructs the exact same string -- it has to match verbatim.
 *
 * @param path [string] endpoint words, e.g. "affiliate code"
 * @param address [string] the wallet address of the signer
 * @param date [Date] the timestamp also sent in the `delphi-ts` header
 */
export const getDelphiMessage = (
  path: string,
  address: string,
  date: Date
): string => {
  return `${path}, ${address.toLowerCase()}, ${getRFC3339String(date)}`;
};

export const getTimeElapsedString = (date: string): string => {
  const inputDate = dayjs(date);
  const elapsedMs = Date.now() / 1000 - inputDate.unix();

  // More then 1 year ago
  let interval = elapsedMs / 31536000;
  if (interval > 1) {
    return inputDate.format("MMM DD, YYYY");
  }

  // More than a day
  interval = elapsedMs / 86400;
  if (interval > 1) {
    return inputDate.format("MMM DD");
  }

  interval = elapsedMs / 3600;
  if (interval > 1) {
    return `${Math.floor(interval)}h`;
  }

  interval = elapsedMs / 60;
  if (interval > 1) {
    return `${Math.floor(interval)}m`;
  }

  return "just now";
};
