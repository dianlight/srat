import pkg from '../package.json'


export function Footer() {
    return <>
        <div className="container">
            <div className="row">
                <div className="col l6 s12">
                    Made by <a className="orange-text text-lighten-3" href="http://materializecss.com">Materialize</a>
                </div>
                <div className="col l6 s12">
                    &copy; 2024 {pkg.author.name}
                </div>
            </div >
        </div >
        < div className="footer-copyright">
        </div >
    </>
}