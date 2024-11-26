import * as ReactDOM from 'react-dom/client';
import { Component } from "./Component.tsx"
import "materialize-css"
import "./css/style.css"
import "materialize-css/dist/css/materialize.min.css"
//import "./index.html"


const root = ReactDOM.createRoot(document.getElementById('root')!);
root.render(<Component message="Sup!5" />)
