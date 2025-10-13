// Node-compatible test setup (CJS) — side effects only
// Mirrors the TypeScript test setup so the Node wrapper can preload test globals
const { Window } = require('happy-dom');

const win = new Window({
    settings: {
        enableJavaScriptEvaluation: true,
        suppressCodeGenerationFromStringsWarning: true
    }
});
global.window = win;
global.document = win.document;
global.HTMLElement = win.HTMLElement;
global.localStorage = win.localStorage;

process.env = process.env || {};
process.env.APIURL = process.env.APIURL || 'http://localhost:8080';

// no exports — this file is purely for side-effects
