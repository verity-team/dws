import PhotoOutlinedIcon from "@mui/icons-material/PhotoOutlined";
import { ReactElement } from "react";

interface MemeToolbarProps {
  onImageBtnClick: () => void;
  onPostBtnClick: () => void;
  canSubmit: boolean;
}

const MemeInputToolbar = ({
  canSubmit,
  onImageBtnClick,
  onPostBtnClick,
}: MemeToolbarProps): ReactElement<MemeToolbarProps> => {
  return (
    <div className="flex items-center justify-between">
      <div onClick={onImageBtnClick} className="cursor-pointer">
        <PhotoOutlinedIcon className="text-2xl text-blue-400" />
      </div>
      <button
        className="px-6 py-2 rounded-2xl bg-red-500 hover:bg-red-600 disabled:bg-red-300 disabled:hover:bg-red-300 text-white disabled:cursor-not-allowed"
        disabled={!canSubmit}
        onClick={onPostBtnClick}
      >
        Post
      </button>
    </div>
  );
};

export default MemeInputToolbar;
