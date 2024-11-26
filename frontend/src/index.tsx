import * as ReactDOM from 'react-dom/client';
import { Page } from "./Page.tsx"
import { NavBar } from './NavBar.tsx';
import "materialize-css"
import "./css/style.css"
import "materialize-css/dist/css/materialize.min.css"
import { Footer } from './Footer.tsx';
import M from 'materialize-css'

const root = ReactDOM.createRoot(document.getElementById('root')!);
//root.render(<Page message="Sup!5" />)
root.render(<Page />)

const navbar = ReactDOM.createRoot(document.getElementById('navbar')!)
navbar.render(<NavBar />)

const footer = ReactDOM.createRoot(document.getElementById('footer')!)
footer.render(<Footer />)


document.addEventListener("DOMContentLoaded", function (event) {
    M.AutoInit();
    //    $('.tabs').tabs();
})