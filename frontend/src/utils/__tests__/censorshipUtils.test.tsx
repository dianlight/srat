/**
 * Tests for censorshipUtils
 * Tests JSON and INI-style censoring of sensitive data
 */
import "../../../test/setup";
import { describe, it, expect } from "bun:test";
import {
    censorPlainText,
    isSensitiveField,
    censorValue,
    SENSITIVE_KEYWORDS,
} from "../censorshipUtils";

describe("censorshipUtils", () => {
    describe("isSensitiveField", () => {
        it("should identify sensitive field names", () => {
            expect(isSensitiveField("password")).toBe(true);
            expect(isSensitiveField("API_KEY")).toBe(true);
            expect(isSensitiveField("secret_token")).toBe(true);
            expect(isSensitiveField("Authorization")).toBe(true);
        });

        it("should return false for non-sensitive fields", () => {
            expect(isSensitiveField("username")).toBe(false);
            expect(isSensitiveField("hostname")).toBe(false);
            expect(isSensitiveField("port")).toBe(false);
        });

        it("should be case-insensitive", () => {
            expect(isSensitiveField("PASSWORD")).toBe(true);
            expect(isSensitiveField("PaSsWoRd")).toBe(true);
        });
    });

    describe("censorValue", () => {
        it("should censor values with lock emoji", () => {
            const result = censorValue("mysecret");
            expect(result).toContain("🔒");
        });

        it("should cap emoji count at 8", () => {
            const result = censorValue("verylongsecretthatexceedseightcharacters");
            const emojiCount = (result.match(/🔒/g) || []).length;
            expect(emojiCount).toBeLessThanOrEqual(8);
        });
    });

    describe("censorPlainText", () => {
        describe("INI-style key-value pairs", () => {
            it("should censor INI-style password fields", () => {
                const input = "password = mySecretPassword123";
                const result = censorPlainText(input);
                expect(result).toContain("password");
                expect(result).toContain("🔒");
                expect(result).not.toContain("mySecretPassword123");
            });

            it("should censor with colon separator", () => {
                const input = "api_key: supersecretkey";
                const result = censorPlainText(input);
                expect(result).toContain("api_key");
                expect(result).toContain("🔒");
                expect(result).not.toContain("supersecretkey");
            });

            it("should preserve indentation", () => {
                const input = "  password = secret123";
                const result = censorPlainText(input);
                expect(result.startsWith("  ")).toBe(true);
            });

            it("should not censor non-sensitive fields", () => {
                const input = "username = johndoe";
                const result = censorPlainText(input);
                expect(result).toBe(input);
            });
        });

        describe("JSON-style key-value pairs in comments", () => {
            it("should censor JSON password field in comments", () => {
                // First 6 lines from unsaved editor (Untitled-2) - simplified JSON
                const input = `# DEBUG: {"CurrentFile":"","allow_guest":true,"password":"JfN0v$Iv2*KYmm1Q"}`;
                const result = censorPlainText(input);
                expect(result).toContain('"password"');
                expect(result).toContain("🔒");
                expect(result).not.toContain("JfN0v$Iv2*KYmm1Q");
            });

            it("should censor mqtt_password in JSON", () => {
                const input = `# DEBUG: {"mqtt_password":"my_mqtt_pass_123"}`;
                const result = censorPlainText(input);
                expect(result).toContain('"mqtt_password"');
                expect(result).toContain("🔒");
                expect(result).not.toContain("my_mqtt_pass_123");
            });

            it("should censor ssh_private_key in JSON", () => {
                const input = `{"ssh_private_key":"-----BEGIN RSA PRIVATE KEY-----..."}`;
                const result = censorPlainText(input);
                expect(result).toContain('"ssh_private_key"');
                expect(result).toContain("🔒");
            });

            it("should handle multiple sensitive fields in one JSON line", () => {
                const input = `# CONFIG: {"password":"pass123","api_key":"key456","username":"user"}`;
                const result = censorPlainText(input);
                expect(result).toContain('"password"');
                expect(result).toContain('"api_key"');
                expect(result).toContain('"username"'); // not censored
                // Should have exactly 2 censored values (password and api_key)
                const emojiCount = (result.match(/🔒/g) || []).length;
                expect(emojiCount).toBeGreaterThanOrEqual(2); // At least password and api_key
                expect(result).not.toContain("pass123");
                expect(result).not.toContain("key456");
            });

            it("should preserve non-sensitive JSON fields", () => {
                const input = `{"hostname":"myhost","allow_guest":true}`;
                const result = censorPlainText(input);
                expect(result).toBe(input);
            });

            it("should handle single quotes in JSON", () => {
                const input = `{'password':'secret123'}`;
                const result = censorPlainText(input);
                expect(result).toContain('"password"'); // Normalized to double quotes
                expect(result).toContain("🔒");
                expect(result).not.toContain("secret123");
            });

            it("should censor mqtt_username when it contains sensitive keywords", () => {
                const input = `{"mqtt_username":"admin","mqtt_password":"secret_pass"}`;
                const result = censorPlainText(input);
                // mqtt_username doesn't contain sensitive keywords, so not censored
                expect(result).toContain("admin");
                // But password should be censored
                expect(result).toContain("🔒");
            });
        });

        describe("Complex real-world examples from Samba config", () => {
            it("should censor password in full JSON DEBUG line", () => {
                // Based on first line from unsaved editor
                const input = `# DEBUG: {"CurrentFile":"","allow_guest":true,"password":"JfN0v$Iv2*KYmm1Q","username":"homeassistant"}`;
                const result = censorPlainText(input);
                expect(result).not.toContain("JfN0v$Iv2*KYmm1Q");
                expect(result).toContain("🔒");
                expect(result).toContain('"username":"homeassistant"'); // not censored
            });

            it("should censor password in nested objects (other_users array)", () => {
                const input = `{"other_users":[{"password":"pippolo123","username":"prova"}],"password":"mainpass123"}`;
                const result = censorPlainText(input);
                // Both passwords should be censored
                expect(result).not.toContain("pippolo123");
                expect(result).not.toContain("mainpass123");
                expect((result.match(/🔒/g) || []).length).toBeGreaterThanOrEqual(2);
                expect(result).toContain('"username":"prova"'); // username not censored
            });

            it("should censor mqtt_password field", () => {
                const input = `{"mqtt_enable":true,"mqtt_password":"mqtt_secret_123","mqtt_username":"admin"}`;
                const result = censorPlainText(input);
                expect(result).not.toContain("mqtt_secret_123");
                expect(result).toContain("🔒");
                expect(result).toContain('"mqtt_username":"admin"'); // username not censored
            });

            it("should censor ssh_private_key field", () => {
                const input = `{"medialibrary":{"enable":false,"ssh_private_key":"-----BEGIN RSA PRIVATE KEY-----"}}`;
                const result = censorPlainText(input);
                expect(result).not.toContain("-----BEGIN RSA PRIVATE KEY-----");
                expect(result).toContain("🔒");
            });

            it("should handle INI section with password in comment above it", () => {
                const input = `# DEBUG: {"password":"secret123"}
[CAROLA]
   path = /mnt/Carola
   password = section_secret`;
                const result = censorPlainText(input);
                expect(result.split("\n")[0]).toContain("🔒"); // JSON password censored
                expect(result).not.toContain("secret123");
                expect(result).not.toContain("section_secret");
            });

            it("should not alter non-password debug comments", () => {
                const input = `# DEBUG: Log Level: debug`;
                const result = censorPlainText(input);
                expect(result).toBe(input);
            });

            it("should censor multiple sensitive fields in deeply nested JSON", () => {
                const input = `{"level1":{"level2":{"password":"deep_pass","api_key":"deep_key","username":"user"}}}`;
                const result = censorPlainText(input);
                expect(result).not.toContain("deep_pass");
                expect(result).not.toContain("deep_key");
                expect(result).toContain('"username":"user"'); // not sensitive
                const emojiCount = (result.match(/🔒/g) || []).length;
                expect(emojiCount).toBeGreaterThanOrEqual(2);
            });

            it("should handle values with special characters", () => {
                const input = `password = "p@$$w0rd!&*%^"`;
                const result = censorPlainText(input);
                expect(result).toContain("🔒");
                expect(result).not.toContain("p@$$w0rd!&*%^");
            });

            it("should not censor in middle of words", () => {
                const input = `mypassword = value`;
                const result = censorPlainText(input);
                expect(result).toContain("🔒"); // 'password' is part of 'mypassword'
            });

            it("should handle consecutive sensitive fields", () => {
                const input = `password = secret1
api_key = secret2
token = secret3`;
                const result = censorPlainText(input);
                const lines = result.split("\n");
                expect(lines[0]).toContain("🔒");
                expect(lines[1]).toContain("🔒");
                expect(lines[2]).toContain("🔒");
            });
        });
    });
});
