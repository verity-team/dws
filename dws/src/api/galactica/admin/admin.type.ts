import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";

export interface ChangeMemeStatusRequest {
  status: MemeUploadStatus;
}

export interface SignInWithCredentialsRequest {
  username: string;
  password: string;
}
