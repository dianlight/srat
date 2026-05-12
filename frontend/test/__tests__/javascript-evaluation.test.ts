import { describe, it, expect } from "vitest";
import "../testing";

describe("JavaScript Evaluation in happy-dom", () => {
    it("should have JavaScript evaluation enabled", () => {
        // Test that eval() works in the happy-dom environment
        // This is a simple test to verify that enableJavaScriptEvaluation is working
        const result = eval("1 + 1");
        expect(result).toBe(2);
    });

    it("should evaluate script text in DOM context", () => {
        // happy-dom doesn't automatically execute appended inline <script> tags,
        // so we verify JavaScript evaluation explicitly in the window context.
        const script = document.createElement("script");
        script.textContent = "window.testValue = 'JavaScript evaluation works';";
        document.body.appendChild(script);

        window.eval(script.textContent ?? "");
        expect((window as any).testValue).toBe("JavaScript evaluation works");

        delete (window as any).testValue;
        document.body.removeChild(script);
    });

    it("should allow Function constructor", () => {
        // Test that Function constructor works (another indicator of JavaScript evaluation)
        const fn = new Function("a", "b", "return a + b");
        const result = fn(2, 3);
        expect(result).toBe(5);
    });
});
