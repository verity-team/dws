export const getExponentialWaitTime = (
  initialTime: number,
  time: number
): number => {
  return initialTime * Math.pow(2, time);
};

export const sleep = async (ms: number): Promise<void> => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};
