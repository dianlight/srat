import { describe, it, expect } from "vitest";

describe("Browser mode detection", () => {
  it("checks various detection methods", () => {
    console.log('=== Environment Check ===');
    console.log('typeof window:', typeof window);
    console.log('typeof document:', typeof document);
    console.log('typeof navigator:', typeof navigator);
    console.log('typeof process:', typeof process);
    
    if (typeof process !== 'undefined') {
      console.log('process.env:', JSON.stringify(process.env));
    }
    
    // Check for Vitest browser mode indicators
    console.log('--- Vitest specific ---');
    // @ts-ignore
    console.log('import.meta.env:', JSON.stringify(import.meta.env));
    
    // Check if we're in a browser-like environment
    const isBrowserLike = typeof window !== 'undefined' && 
                         typeof document !== 'undefined' &&
                         typeof navigator !== 'undefined';
    console.log('isBrowserLike:', isBrowserLike);
    
    // Check for jsdom vs real browser
    const hasJsdom = (typeof navigator !== 'undefined' && 
                     navigator.userAgent.includes('jsdom'));
    console.log('hasJsdom:', hasJsdom);
    
    // Check for Playwright/Vitest browser
    const isPlaywright = (typeof navigator !== 'undefined' && 
                         navigator.userAgent.includes('Playwright'));
    console.log('isPlaywright:', isPlaywright);
    
    // Alternative: check if Vitest browser globals are available
    // @ts-ignore
    const hasVitestBrowser = typeof page !== 'undefined';
    console.log('hasVitestBrowser (page global):', hasVitestBrowser);
    
    expect(true).toBe(true);
  });
});