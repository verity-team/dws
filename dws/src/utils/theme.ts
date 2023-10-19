import { Shadows, createTheme } from "@mui/material";

export const theme = createTheme({
  shadows: Array(25).fill("none") as Shadows,
});
