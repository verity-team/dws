import { useCallback, useState } from "react";

export const useToggle = (init: boolean = false) => {
  const [isOpen, setIsOpen] = useState(init);

  const open = useCallback(() => {
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
  }, []);

  const toggle = useCallback(() => {
    setIsOpen((open) => !open);
  }, []);

  return { isOpen, setIsOpen, open, close, toggle };
};
