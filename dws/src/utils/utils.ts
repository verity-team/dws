export const getExponentialWaitTime = (
  initialTime: number,
  time: number
): number => {
  return initialTime * Math.pow(2, time);
};
