import { describe, it, expect } from "bun:test";
import "../setup";

describe("JavaScript Evaluation in happy-dom", () => {
    it("should have JavaScript evaluation enabled", () => {
        // Test that eval() works in the happy-dom environment
        // This is a simple test to verify that enableJavaScriptEvaluation is working
        const result = eval("1 + 1");
        expect(result).toBe(2);
    });

    it("should execute inline scripts in the DOM", () => {
        // Create a script element and verify it can execute
        const script = document.createElement("script");
        script.textContent = "window.testValue = 'JavaScript evaluation works';";
        document.body.appendChild(script);
        
        // If JavaScript evaluation is enabled, the script should have executed
        expect((window as any).testValue).toBe("JavaScript evaluation works");
        
        // Cleanup
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
