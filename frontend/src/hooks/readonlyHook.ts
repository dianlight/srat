import { useContext } from "react";
import { ModeContext } from "../Contexts";

export function useReadOnly() {
    const mode = useContext(ModeContext);
    return mode.read_only || false;
}