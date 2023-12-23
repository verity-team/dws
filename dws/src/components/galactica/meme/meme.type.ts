export type MemeUploadStatus = "PENDING" | "APPROVED" | "DENIED";

export interface MemeFilter {
  status?: MemeUploadStatus;
}
