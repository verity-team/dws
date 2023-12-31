import { MemeUploadStatus } from "@/components/galactica/meme/meme.type";
import { Tooltip } from "@mui/material";

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
  if (memeStatus === "APPROVED" || memeStatus === "DENIED") {
    return (
      <>
        <div className="mt-12 text-2xl">Actions:</div>
        <div className="w-full py-2 text-white space-x-4">
          <Tooltip title="Change this meme's status to PENDING" placement="top">
            <button
              className="px-4 py-2 bg-blue-500 hover:bg-blue-600 rounded-lg"
              onClick={onRevert}
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
      <div className="mt-12 text-xl">Review:</div>
      <div className="w-full py-2 text-white space-x-4 flex items-center justify-start">
        <button
          className="px-6 py-2 bg-red-500 hover:bg-red-600 rounded-lg"
          onClick={onDeny}
        >
          Decline
        </button>
        <button
          className="px-6 py-2 bg-green-500 hover:bg-green-600 rounded-lg"
          onClick={onApprove}
        >
          Approve
        </button>
      </div>
    </>
  );
};

export default AdminToolbar;
