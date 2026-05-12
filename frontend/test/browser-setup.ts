/**
 * Vitest browser test setup entrypoint.
 */

import "./common-setup";
import "./msw-browser-setup";

export { createTestStore } from "./common-setup";
export {
	getMswWorker as getMswServer,
	mswWorker as mswServer,
	withTestHandlers,
} from "./msw-browser-setup";
