/**
 * Default-export variant of the dev-inspector plugin.
 * Use this when registering via bunfig.toml [serve.static] plugins,
 * or when passing directly to Bun.build({ plugins: [...] }).
 *
 * For the Bun.serve() JS API, use inspector-plugin.ts with Bun.plugin() instead.
 */
import { devInspectorPlugin } from "./inspector-plugin";
export default devInspectorPlugin;
