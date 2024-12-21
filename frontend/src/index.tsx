import * as ReactDOM from 'react-dom/client';
import { Page } from "./Page.tsx"
import "../node_modules/@materializecss/materialize/dist/css/materialize.min.css.map"
import "../node_modules/@materializecss/materialize/dist/css/materialize.min.css"
import "./css/style.css"
import { ErrorBoundaryContext } from 'react-use-error-boundary';

const root = ReactDOM.createRoot(document.getElementById('root')!);
root.render(<ErrorBoundaryContext><Page /></ErrorBoundaryContext>)
