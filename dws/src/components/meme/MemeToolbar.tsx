import PhotoOutlinedIcon from "@mui/icons-material/PhotoOutlined";
import { ReactElement } from "react";

interface MemeToolbarProps {
  onImageBtnClick: () => void;
  onPostBtnClick: () => void;
  canSubmit: boolean;
}

const MemeToolbar = ({
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
        className="px-6 py-2 rounded-2xl bg-blue-300 hover:bg-blue-500 text-white disabled:cursor-not-allowed disabled:hover:bg-blue-300"
        disabled={!canSubmit}
        onClick={onPostBtnClick}
      >
        Post
      </button>
    </div>
  );
};

export default MemeToolbar;
