import DataObjectIcon from "@mui/icons-material/DataObject"; // For camelCase
import KeyboardCapslockIcon from "@mui/icons-material/KeyboardCapslock"; // For UPPERCASE
import RemoveIcon from "@mui/icons-material/Remove"; // Import RemoveIcon for kebab-case
import TextDecreaseIcon from "@mui/icons-material/TextDecrease"; // For lowercase
import type { SvgIconTypeMap } from "@mui/material";
import type { OverridableComponent } from "@mui/material/OverridableComponent";
import { CasingStyle } from "./types";

// Helper function to extract basename from a path
export function getPathBaseName(path: string): string {
	if (!path) return "";
	// Remove trailing slashes to correctly get the last segment
	const p = path.replace(/\/+$/, "");
	const lastSegment = p.substring(p.lastIndexOf("/") + 1);
	// Return empty string if lastSegment is empty (e.g. path was just "/")
	return lastSegment === "" && p === "/" ? "" : lastSegment;
}

// Helper function to sanitize a string for use as a Windows share name and convert to uppercase
export function sanitizeAndUppercaseShareName(name: string): string {
	if (!name) return "";
	// Replace invalid characters (/:*?"<>|-) and whitespace with an underscore, then convert to uppercase
	return name.replace(/[/:*?"<>|\s-]+/g, "_").toUpperCase();
}

// --- Veto File Entry Validation Helper ---
// Matches a valid Samba veto file entry:
// - Not empty
// - Does not contain '/' (as it's a separator for the list in smb.conf)
// - Does not contain null byte '\0'
export const VETO_FILE_ENTRY_REGEX = /^[^/\0]+$/;

export function isValidVetoFileEntry(entry: string): boolean {
	if (typeof entry !== "string") return false;
	return VETO_FILE_ENTRY_REGEX.test(entry);
}

// --- Casing Styles and Helpers ---

export const casingCycleOrder: CasingStyle[] = [
	CasingStyle.UPPERCASE,
	CasingStyle.LOWERCASE,
	CasingStyle.CAMELCASE,
	CasingStyle.KEBABCASE,
];

// Helper to split words based on common separators and camelCase transitions
const splitWords = (str: string): string[] => {
	if (!str) return [];
	const s1 = str.replace(/([a-z0-9])([A-Z])/g, "$1 $2"); // myWord -> my Word
	const s2 = s1.replace(/([A-Z])([A-Z][a-z])/g, "$1 $2"); // ABBRWord -> ABBR Word
	const s3 = s2.replace(/[_-]+/g, " "); // Replace _ and - with space
	return s3.split(/\s+/).filter(Boolean); // Split by space and remove empty parts
};

export const toCamelCase = (str: string): string => {
	const words = splitWords(str);
	if (words.length === 0) return "";
	return words
		.map((word, index) =>
			index === 0
				? word.toLowerCase()
				: word.charAt(0).toUpperCase() + word.slice(1).toLowerCase(),
		)
		.join("");
};

export const toKebabCase = (str: string): string => {
	const words = splitWords(str);
	if (words.length === 0) return "";
	return words.map((word) => word.toLowerCase()).join("_");
};

const casingStyleToIconMap: Record<
	CasingStyle,
	OverridableComponent<SvgIconTypeMap<Record<string, never>, "svg">>
> = {
	[CasingStyle.UPPERCASE]: KeyboardCapslockIcon,
	[CasingStyle.LOWERCASE]: TextDecreaseIcon,
	[CasingStyle.CAMELCASE]: DataObjectIcon,
	[CasingStyle.KEBABCASE]: RemoveIcon,
};

export const getCasingIcon = (
	style: CasingStyle,
): OverridableComponent<SvgIconTypeMap<Record<string, never>, "svg">> => {
	return casingStyleToIconMap[style] || KeyboardCapslockIcon; // Default to UPPERCASE icon if not found
};
