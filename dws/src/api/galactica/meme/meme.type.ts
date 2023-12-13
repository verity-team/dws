export interface MemeUploadDTO {
  userId: string;
  meme: File;
  caption: string;
}

export interface MemeUpload {
  fileId: string;
  userId: string;
  caption: string;
  lang: string;
  tags: string[];
  createdAt: Date;
  updatedAt: Date;
}
