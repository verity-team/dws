import { Dialog, DialogTitle, DialogContent } from "@mui/material";
import { ReactElement } from "react";

interface ThankDialogProps {
  isOpen: boolean;
  handleClose: () => void;
}

const ThankDialog = ({
  isOpen,
  handleClose,
}: ThankDialogProps): ReactElement<ThankDialogProps> => {
  return (
    <Dialog
      open={isOpen}
      onClose={handleClose}
      fullWidth={true}
      maxWidth="sm"
      className="rounded-lg"
    >
      <DialogTitle className="text-2xl">Thanks for your support 🎉</DialogTitle>
      <DialogContent className="font-sans w-full">
        Your donation will be confirmed on the blockchain shortly and shown
        below.
      </DialogContent>
    </Dialog>
  );
};

export default ThankDialog;
