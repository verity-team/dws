import { useToggle } from "@/hooks/utils/useToggle";
import { Menu, MenuItem } from "@mui/material";
import ShareIcon from "@mui/icons-material/Share";
import LinkIcon from "@mui/icons-material/Link";
import { TwitterShareButton, TwitterIcon } from "react-share";
import { ReactElement, SyntheticEvent, useState } from "react";
import { Nullable } from "@/utils";
import toast from "react-hot-toast";

interface ShareButtonProps {
  memeId: string;
}

const ShareButton = ({
  memeId,
}: ShareButtonProps): ReactElement<ShareButtonProps> => {
  const {
    isOpen,
    open: openShareDropdown,
    close: closeShareDropdown,
  } = useToggle(false);

  const [anchorEl, setAnchorEl] = useState<Nullable<HTMLButtonElement>>(null);

  const handleShareClick = (event: SyntheticEvent<HTMLButtonElement>) => {
    event.stopPropagation();

    setAnchorEl(event.currentTarget);
    openShareDropdown();
  };

  const handleCopyClick = () => {
    navigator.clipboard.writeText(`${window.location.href}/${memeId}`);
    toast("Copied to clipboard", { icon: "📋" });
    closeShareDropdown();
  };

  return (
    <>
      <button
        type="button"
        className="flex items-center justify-center px-4 py-2 rounded-lg hover:bg-slate-200 m-1"
        onClick={handleShareClick}
      >
        <ShareIcon className="text-2xl" />
        <div className="ml-1.5 text-lg font-changa">Share</div>
      </button>
      <Menu
        anchorEl={anchorEl}
        open={isOpen}
        onClose={closeShareDropdown}
        anchorOrigin={{
          vertical: "bottom",
          horizontal: "left",
        }}
        className="space-y-4"
        PopoverClasses={{
          paper: "w-[200px] max-w-full",
        }}
      >
        <div className="text-lg font-bold px-4">Share to</div>
        <MenuItem>
          <TwitterShareButton
            title="Check out this fresh meme on TruthMemes!"
            url={`${window.location.href}/${memeId}`}
            hashtags={["memes", "truthmemes"]}
            className="w-full"
            onClick={closeShareDropdown}
          >
            <div className="flex items-center">
              <TwitterIcon size={32} round />
              <div className="ml-2">Twitter</div>
            </div>
          </TwitterShareButton>
        </MenuItem>
        <MenuItem>
          <div
            className="flex items-center justify-between"
            onClick={handleCopyClick}
          >
            <div className="flex items-center justify-center p-2 bg-slate-200 rounded-full">
              <LinkIcon className="rotate-45 text-xl" />
            </div>
            <div className="ml-2">Copy link</div>
          </div>
        </MenuItem>
      </Menu>
    </>
  );
};

export default ShareButton;
