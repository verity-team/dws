import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";
import { Tooltip } from "@mui/material";
import { useState } from "react";

interface AdminToolbarProps {
  memeStatus: MemeUploadStatus;
  onRevert: () => void;
  onApprove: () => void;
  onDeny: () => void;
}

const AdminToolbar = ({
  memeStatus,
  onRevert,
  onDeny,
  onApprove,
}: AdminToolbarProps) => {
  const [innerStatus, setInnerStatus] = useState(memeStatus);

  const handleRevert = () => {
    setInnerStatus("PENDING");
    onRevert();
  };

  const handleApprove = () => {
    setInnerStatus("APPROVED");
    onApprove();
  };

  const handleDeny = () => {
    setInnerStatus("DENIED");
    onDeny();
  };

  if (innerStatus === "APPROVED" && memeStatus === "PENDING") {
    return (
      <>
        <div className="mt-4 text-2xl">You have approved this meme</div>
        <div className="w-full py-2 text-white space-x-4">
          <Tooltip title="Change this meme's status to PENDING" placement="top">
            <button
              className="px-4 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg"
              onClick={handleRevert}
            >
              Revert?
            </button>
          </Tooltip>
        </div>
      </>
    );
  }

  if (innerStatus === "DENIED" && memeStatus === "PENDING") {
    return (
      <>
        <div className="mt-4 text-2xl">You have denied this meme</div>
        <div className="w-full py-2 text-white space-x-4">
          <Tooltip title="Change this meme's status to PENDING" placement="top">
            <button
              className="px-4 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg"
              onClick={handleRevert}
            >
              Revert?
            </button>
          </Tooltip>
        </div>
      </>
    );
  }

  if (
    (innerStatus !== memeStatus && memeStatus === "APPROVED") ||
    memeStatus === "DENIED"
  ) {
    return (
      <>
        <div className="mt-4 text-xl">Actions:</div>
        <div className="w-full py-2 text-white space-x-4">
          <Tooltip title="Check this meme in the PENDING tab" placement="top">
            <button
              className="px-4 py-2 bg-gray-600 rounded-lg disabled:cursor-not-allowed"
              disabled
            >
              Reverted
            </button>
          </Tooltip>
        </div>
      </>
    );
  }

  if (memeStatus === "APPROVED" || memeStatus === "DENIED") {
    return (
      <>
        <div className="mt-4 text-xl">Actions:</div>
        <div className="w-full py-2 text-white space-x-4">
          <Tooltip title="Change this meme's status to PENDING" placement="top">
            <button
              className="px-4 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg"
              onClick={handleRevert}
            >
              Revert
            </button>
          </Tooltip>
        </div>
      </>
    );
  }

  return (
    <>
      <div className="mt-4 text-xl">Review:</div>
      <div className="w-full py-2 text-white space-x-4 flex items-center justify-start">
        <button
          className="px-6 py-2 bg-red-500 hover:bg-red-600 rounded-lg"
          onClick={handleDeny}
        >
          Decline
        </button>
        <button
          className="px-6 py-2 bg-green-500 hover:bg-green-600 rounded-lg"
          onClick={handleApprove}
        >
          Approve
        </button>
      </div>
    </>
  );
};

export default AdminToolbar;
