export * from "./campaign.type";
export * from "./userData.type";

export interface FailedResponse {
  code: string;
  message: string;
}

export interface CustomError extends Error {
  info: FailedResponse;
  status: number;
}
