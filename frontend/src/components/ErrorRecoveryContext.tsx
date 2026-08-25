import { createContext, useContext } from "react";

interface ErrorRecoveryContextValue {
  /** Increments each time the error boundary resets; consumers can key
   *  components off this to force a clean remount after recovery. */
  resetKey: number;
}

const ErrorRecoveryContext = createContext<ErrorRecoveryContextValue>({
  resetKey: 0,
});

export const useErrorRecovery = () => useContext(ErrorRecoveryContext);

export default ErrorRecoveryContext;
