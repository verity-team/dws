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
  createdAt: string;
  updatedAt: string;
}

// For instantly displaying after user upload
export interface OptimisticMemeUpload {
  fileId: string;
  userId: string;
  caption: string;
  createdAt: string;
}
