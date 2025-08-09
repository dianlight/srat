import { useEffect, useRef } from "react";
import {
	type ConsoleErrorCallback,
	registerConsoleErrorCallback,
} from "../devtool/consoleErrorRegistry";

/**
 * React hook to register a console.error callback for the lifetime of a component.
 * The callback is registered on mount and unregistered on unmount.
 */
export function useConsoleErrorCallback(cb: ConsoleErrorCallback) {
	const cbRef = useRef(cb);
	cbRef.current = cb;

	useEffect(() => {
		const unsubscribe = registerConsoleErrorCallback((...args) =>
			cbRef.current(...args),
		);
		return () => unsubscribe();
	}, []);
}
