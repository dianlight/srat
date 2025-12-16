#!/usr/bin/env bun
/**
 * Validator for macro imports
 *
 * This script checks that all imports from the macro directory use the proper
 * import assertion: `with { type: "macro" }`
 *
 * Usage: bun run scripts/validate-macro-imports.js
 */

import fs from "fs";
import path from "path";

// Regex patterns to detect macro imports
const macroImportPattern = /import\s+(?:{[^}]*?}|[a-zA-Z_$][a-zA-Z0-9_$]*)\s+from\s+["']([^"']*(?:\.\/)?macro[^"']*)["'](?!\s*with\s*{\s*type\s*:\s*["']macro["']\s*})/g;

const macroImportWithoutAssertion = /import\s+(?:{[^}]*?}|[a-zA-Z_$][a-zA-Z0-9_$]*(?:\s*,\s*\*\s*as\s+[a-zA-Z_$][a-zA-Z0-9_$]*)?)\s+from\s+["']([^"']*\/macro\/[^"']*)["'](?!\s*with)/g;

// Recursive function to get all TypeScript files
function getAllTypeScriptFiles(dir, fileList = []) {
	const files = fs.readdirSync(dir);

	for (const file of files) {
		const filePath = path.join(dir, file);
		const stat = fs.statSync(filePath);

		if (stat.isDirectory()) {
			// Skip node_modules, __tests__, and dotfiles
			if (!file.startsWith(".") && file !== "node_modules" && file !== "__tests__") {
				getAllTypeScriptFiles(filePath, fileList);
			}
		} else if ((file.endsWith(".ts") || file.endsWith(".tsx")) && file !== "sratApi.ts") {
			fileList.push(filePath);
		}
	}

	return fileList;
}

// Check if import is from macro directory
function isMacroImport(source) {
	return source.includes("/macro/") || source === "./macro" || source === "../macro";
}

// Check if import has the proper assertion (may span multiple lines)
function hasProperAssertion(lines, startIndex) {
	// Join up to 5 lines to check for the assertion across line breaks
	let searchText = lines.slice(startIndex, Math.min(startIndex + 5, lines.length)).join(" ");
	// Match: with { type: "macro" } or with { type: "macro", } (trailing comma allowed)
	return /with\s*{\s*type\s*:\s*["']macro["']\s*,?\s*}/.test(searchText);
}

// Main validation function
function validateMacroImports() {
	const issues = [];
	const frontendDir = process.cwd();
	const srcDir = path.join(frontendDir, "src");

	// Make sure src directory exists
	if (!fs.existsSync(srcDir)) {
		console.error("Error: src directory not found");
		process.exit(1);
	}

	const tsFiles = getAllTypeScriptFiles(srcDir);

	for (const filePath of tsFiles) {
		try {
			const code = fs.readFileSync(filePath, "utf-8");

			// Skip generated files
			if (code.includes("// @generated")) {
				continue;
			}

			const lines = code.split("\n");
			let lineNumber = 0;

			for (let i = 0; i < lines.length; i++) {
				const line = lines[i];
				lineNumber = i + 1;

				// Check for macro imports
				if (line.includes("from") && line.includes("macro")) {
					// Check if line contains import statement
					if (/import\s+[{]?/.test(line)) {
						// Extract the source path
						const sourceMatch = line.match(/from\s+["']([^"']*macro[^"']*)["']/);
						if (sourceMatch) {
							const source = sourceMatch[1];
							// Check if it has proper assertion (may be on next lines)
							if (!hasProperAssertion(lines, i)) {
								const relativePath = path.relative(frontendDir, filePath);
								issues.push({
									file: relativePath,
									line: lineNumber,
									source,
									message: `Macro import "${source}" is missing 'with { type: "macro" }' assertion`,
								});
							}
						}
					}
				}
			}
		} catch (error) {
			console.error(`Error reading ${filePath}:`, error.message);
		}
	}

	return issues;
}

// Run validation
const issues = validateMacroImports();

if (issues.length === 0) {
	console.log("✅ All macro imports are properly validated!");
	process.exit(0);
} else {
	console.error("❌ Found issues with macro imports:\n");
	for (const issue of issues) {
		console.error(`${issue.file}:${issue.line}`);
		console.error(`  ${issue.message}`);
		console.error(`  Import source: ${issue.source}\n`);
	}
	process.exit(1);
}
