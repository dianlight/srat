#!/usr/bin/env node
// Scan test files and prepend import to ensure shared test setup is present
const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..');
const testDir = path.join(root, 'src');
const setupImport = `import '../../../../test/setup';\n`;
const apply = process.argv.includes('--apply');
let missingFiles = [];

function walk(dir) {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            walk(full);
        } else if (/\.test\.(ts|tsx|js|jsx)$/.test(entry.name)) {
            let content = fs.readFileSync(full, 'utf8');
            if (!content.includes("test/setup") && !content.includes("../../../../test/setup")) {
                missingFiles.push(full);
                if (apply) {
                    content = setupImport + content;
                    fs.writeFileSync(full, content, 'utf8');
                    console.log('Prepended setup to', full);
                }
            }
        }
    }
}

walk(testDir);
if (missingFiles.length > 0) {
    console.error(`Found ${missingFiles.length} test files missing the setup import.`);
    if (!apply) {
        console.error('Run with --apply to prepend the import to those files.');
        missingFiles.slice(0, 10).forEach(f => console.error('  ', f));
        process.exitCode = 2;
    }
}
console.log('Done.');
