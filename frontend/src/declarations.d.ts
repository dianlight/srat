// Wildcard module declarations for non-TS assets
// These are needed for tsgo (TypeScript Go) which enforces allowArbitraryExtensions strictly

declare module "*.css" {
    const content: Record<string, string>;
    export default content;
}

declare module "*.ico" {
    const content: string;
    export default content;
}
