<!-- DOCTOC SKIP -->

---

description: 'ReactJS development standards and best practices'
applyTo: '**/\*.jsx,**/*.tsx, \*\*/*.js, **/\*.ts,**/*.css, \*\*/*.scss'

---

# ReactJS Development Instructions

Instructions for building high-quality ReactJS applications with modern patterns, hooks, and best practices following the official React documentation at https://react.dev.

## Project Context

- Latest React version (React 19+)
- TypeScript for type safety (when applicable)
- Functional components with hooks as default
- Follow React's official style guide and best practices
- Use modern build tools (Vite, Create React App, or custom Webpack setup)
- Implement proper component composition and reusability patterns

## Development Standards

### Architecture

- Use functional components with hooks as the primary pattern
- Implement component composition over inheritance
- Organize components by feature or domain for scalability
- Separate presentational and container components clearly
- Use custom hooks for reusable stateful logic
- Implement proper component hierarchies with clear data flow

### TypeScript Integration

- Use TypeScript interfaces for props, state, and component definitions
- Define proper types for event handlers and refs
- Implement generic components where appropriate
- Use strict mode in `tsconfig.json` for type safety
- Leverage React's built-in types (`React.FC`, `React.ComponentProps`, etc.)
- Create union types for component variants and states

### Component Design

- Follow the single responsibility principle for components
- Use descriptive and consistent naming conventions
- Implement proper prop validation with TypeScript or PropTypes
- Design components to be testable and reusable
- Keep components small and focused on a single concern
- Use composition patterns (render props, children as functions)

### State Management

- Use `useState` for local component state
- Implement `useReducer` for complex state logic
- Leverage `useContext` for sharing state across component trees
- Consider external state management (Redux Toolkit, Zustand) for complex applications
- Implement proper state normalization and data structures
- Use React Query or SWR for server state management

### Hooks and Effects

- Use `useEffect` with proper dependency arrays to avoid infinite loops
- Implement cleanup functions in effects to prevent memory leaks
- Use `useMemo` and `useCallback` for performance optimization when needed
- Create custom hooks for reusable stateful logic
- Follow the rules of hooks (only call at the top level)
- Use `useRef` for accessing DOM elements and storing mutable values

### Styling

- Use CSS Modules, Styled Components, or modern CSS-in-JS solutions
- Implement responsive design with mobile-first approach
- Follow BEM methodology or similar naming conventions for CSS classes
- Use CSS custom properties (variables) for theming
- Implement consistent spacing, typography, and color systems
- Ensure accessibility with proper ARIA attributes and semantic HTML

### Performance Optimization

- Use `React.memo` for component memoization when appropriate
- Implement code splitting with `React.lazy` and `Suspense`
- Optimize bundle size with tree shaking and dynamic imports
- Use `useMemo` and `useCallback` judiciously to prevent unnecessary re-renders
- Implement virtual scrolling for large lists
- Profile components with React DevTools to identify performance bottlenecks

### Data Fetching

- Use modern data fetching libraries (React Query, SWR, Apollo Client)
- Implement proper loading, error, and success states
- Handle race conditions and request cancellation
- Use optimistic updates for better user experience
- Implement proper caching strategies
- Handle offline scenarios and network errors gracefully
- **Never use raw `fetch()` for internal API calls** — always use RTK Query via `sratApi`. For imperative/callback use (e.g. inside `useCallback`), lazy hooks are not exported by the codegen; use `sratApi.endpoints.<endpointName>.useLazyQuery()` instead. When the endpoint response is a union type (e.g. `SuccessType | ErrorModel`), add a type guard before using the success branch.

### Error Handling

- Implement Error Boundaries for component-level error handling
- Use proper error states in data fetching
- Implement fallback UI for error scenarios
- Log errors appropriately for debugging
- Handle async errors in effects and event handlers
- Provide meaningful error messages to users

### Forms and Validation

- **`react-hook-form` + `react-hook-form-mui` is the mandatory standard** for all user input in this project — dialogs, modals, and full pages alike. Never use raw `useState` to manage form field values, validation errors, or submit-loading state.
- Wrap all form fields in `<FormContainer>` from `react-hook-form-mui`, passing `formContext` (from `useForm`) and `onSuccess`. The `onSuccess` handler receives validated data and runs async API calls.
- In a MUI `Dialog`, wrap both `DialogContent` **and** `DialogActions` inside `<FormContainer>` so that `type="submit"` buttons inside `DialogActions` correctly trigger form validation and submission.
- Use `PasswordElement` from `react-hook-form-mui` for password fields — it includes a built-in show/hide toggle. Do not re-implement visibility toggle with `useState`.
- Use `TextFieldElement`, `SelectElement`, `AutocompleteElement`, `SwitchElement`, etc. from `react-hook-form-mui` for all inputs. Pass validation via the `rules` prop.
- Cross-field validation (e.g., confirm password) uses `rules={{ validate: (value, formValues) => value === formValues.otherField || "error message" }}`.
- API/server-level errors belong in `setError("root", { message: "..." })` from `formContext`; render via `formState.errors.root?.message`. No manual error `useState` needed.
- `formState.isSubmitting` is automatically `true` while the async `onSuccess` handler runs — never add a manual `isSubmitting` `useState`.
- See `.github/instructions/react-hook-form-mui.instructions.md` for the complete canonical patterns and examples.

### Routing

- Use React Router for client-side routing
- Implement nested routes and route protection
- Handle route parameters and query strings properly
- Implement lazy loading for route-based code splitting
- Use proper navigation patterns and back button handling
- Implement breadcrumbs and navigation state management

### Testing

- Write unit tests for components using React Testing Library
- Test component behavior, not implementation details
- Use Jest for test runner and assertion library
- Implement integration tests for complex component interactions
- Mock external dependencies and API calls appropriately
- Test accessibility features and keyboard navigation
- Reuse `ReadonlyCommandTerminal` for read-only command/log views and preserve semantic channels (`stdout`, `stderr`, `info`) instead of rendering joined strings or labeling all filesystem task notes as info.

### Guided Tour & Overlay (SRAT)

- Treat `frontend/src/utils/TourEvents.ts` as the authoritative source for guided-tour event names and payload contracts.
- Register `TourEvents.on(...)` subscriptions inside `useEffect` and always clean them up on unmount.
- Keep every `*TourStep.tsx` selector aligned with a real `data-tutor="reactour__tab...__step..."` anchor in live page components.
- Prefer stable container anchors for tour selectors; avoid volatile nested nodes that frequently change during UI refactors.
- If a tour `action` mutates UI state (for example selecting a panel), ensure the action makes the target selector available before highlight.
- Add/update focused tests for tour changes (step structure, emitted events, listener cleanup behavior).

### Security

- Sanitize user inputs to prevent XSS attacks
- Validate and escape data before rendering
- Use HTTPS for all external API calls
- Implement proper authentication and authorization patterns
- Avoid storing sensitive data in localStorage or sessionStorage
- Use Content Security Policy (CSP) headers

### Accessibility

- Use semantic HTML elements appropriately
- Implement proper ARIA attributes and roles
- Ensure keyboard navigation works for all interactive elements
- Provide alt text for images and descriptive text for icons
- Implement proper color contrast ratios
- Test with screen readers and accessibility tools

## Implementation Process

1. Plan component architecture and data flow
2. Set up project structure with proper folder organization
3. Define TypeScript interfaces and types
4. Implement core components with proper styling
5. Add state management and data fetching logic
6. Implement routing and navigation
7. Add form handling and validation
8. Implement error handling and loading states
9. Add testing coverage for components and functionality
10. Optimize performance and bundle size
11. Ensure accessibility compliance
12. Add documentation and code comments

## Additional Guidelines

- Follow React's naming conventions (PascalCase for components, camelCase for functions)
- Use meaningful commit messages and maintain clean git history
- Implement proper code splitting and lazy loading strategies
- Document complex components and custom hooks with JSDoc
- Use ESLint and Prettier for consistent code formatting
- Keep dependencies up to date and audit for security vulnerabilities
- Implement proper environment configuration for different deployment stages
- Use React Developer Tools for debugging and performance analysis

## Common Patterns

- Higher-Order Components (HOCs) for cross-cutting concerns
- Render props pattern for component composition
- Compound components for related functionality
- Provider pattern for context-based state sharing
- Container/Presentational component separation
- Custom hooks for reusable logic extraction
