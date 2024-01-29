import { useState } from "react";
import { MemeUploadStatus } from "../../meme/meme.type";

const AdminMemeTimeline = () => {
  // Default for admin to see pending memes on page load
  const [selectedStatus, setSelectedStatus] =
    useState<MemeUploadStatus>("PENDING");

  return <div></div>;
};

export default AdminMemeTimeline;
