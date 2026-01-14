import "../../../test/setup";
import { describe, it, expect } from "bun:test";
import { 
	isSensitiveField, 
	censorValue, 
	censorPlainText,
	objectToPlainText,
	objectToMarkdown 
} from "../censorshipUtils";

describe("Censorship Utils", () => {
	describe("isSensitiveField", () => {
		it("detects password-related fields", () => {
			expect(isSensitiveField("password")).toBe(true);
			expect(isSensitiveField("user_password")).toBe(true);
			expect(isSensitiveField("PASSWORD")).toBe(true);
			expect(isSensitiveField("pass")).toBe(true);
		});

		it("detects token-related fields", () => {
			expect(isSensitiveField("api_token")).toBe(true);
			expect(isSensitiveField("access_token")).toBe(true);
			expect(isSensitiveField("bearer_token")).toBe(true);
		});

		it("detects key-related fields", () => {
			expect(isSensitiveField("api_key")).toBe(true);
			expect(isSensitiveField("secret_key")).toBe(true);
			expect(isSensitiveField("private_key")).toBe(true);
		});

		it("does not detect non-sensitive fields", () => {
			expect(isSensitiveField("username")).toBe(false);
			expect(isSensitiveField("email")).toBe(false);
			expect(isSensitiveField("name")).toBe(false);
		});
	});

	describe("censorValue", () => {
		it("censors string values with lock emoji", () => {
			const result = censorValue("mysecret");
			expect(result).toContain("🔒");
			// Should return multiple lock emojis (8 max based on string length)
			expect(result.length).toBeGreaterThan(0);
			expect(result).toBe("🔒".repeat(8)); // "mysecret" is 8 chars
		});

		it("censors non-string values", () => {
			const result = censorValue(12345);
			expect(result).toContain("🔒");
			// Numbers are converted to string, so 12345 = 5 chars = 5 emojis
			expect(result).toBe("🔒".repeat(5));
		});
	});

	describe("censorPlainText", () => {
		it("censors sensitive key-value pairs in INI format", () => {
			const input = `
[section]
username = admin
password = secret123
api_key=myapikey
normal_value = visible
`;
			const result = censorPlainText(input);
			
			expect(result).toContain("username = admin");
			expect(result).toContain("🔒");
			expect(result).not.toContain("secret123");
			expect(result).not.toContain("myapikey");
			expect(result).toContain("normal_value = visible");
		});

		it("preserves non-key-value lines", () => {
			const input = `
[section]
# This is a comment
password = secret
`;
			const result = censorPlainText(input);
			
			expect(result).toContain("[section]");
			expect(result).toContain("# This is a comment");
		});

		it("handles colon separators", () => {
			const input = "password: mysecret";
			const result = censorPlainText(input);
			
			expect(result).toContain("🔒");
			expect(result).not.toContain("mysecret");
		});
	});

	describe("objectToPlainText", () => {
		it("converts object to plain text with censorship", () => {
			const obj = {
				username: "admin",
				password: "secret",
				active: true
			};
			
			const result = objectToPlainText(obj);
			
			expect(result).toContain("username: admin");
			expect(result).toContain("🔒");
			expect(result).toContain("censored");
			expect(result).not.toContain("secret");
			expect(result).toContain("active: Yes");
		});
	});

	describe("objectToMarkdown", () => {
		it("converts object to markdown with censorship", () => {
			const obj = {
				username: "admin",
				password: "secret"
			};
			
			const result = objectToMarkdown(obj);
			
			expect(result).toContain("**username**");
			expect(result).toContain("`admin`");
			expect(result).toContain("**password**");
			expect(result).toContain("🔒");
			expect(result).toContain("censored");
			expect(result).not.toContain("secret");
		});
	});
});
