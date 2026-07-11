// Wildcard module declarations for non-TS assets
// These are needed for the TypeScript 7 (Go-based) compiler which enforces allowArbitraryExtensions strictly

declare module "*.css" {
  const content: Record<string, string>;
  export default content;
}

declare module "*.ico" {
  const content: string;
  export default content;
}
