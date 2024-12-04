import pkg from '../package.json';
import { getGitCommitHash } from './macro/getGitCommitHash.ts' with { type: 'macro' };


export function Footer() {
    return <> <div className="container">
        {/*
        <div className="row">
            <div className="l6 s12">
                <h5>Footer Content</h5>
                <p>You can use rows and columns here to organize your footer content.</p>
            </div>
            <div className="l4 offset-l8 s12">
                <h5>Links</h5>
                <ul>
                    <li><a href="#!">Link 1</a></li>
                    <li><a href="#!">Link 2</a></li>
                    <li><a href="#!">Link 3</a></li>
                    <li><a href="#!">Link 4</a></li>
                </ul>
            </div>
        </div>
        */}
    </div>
        <div className="footer-copyright">
            <div className="container">
                © 2014 Copyright {pkg.author.name}
                <a className="right" href={pkg.repository.url + "/commit/" + getGitCommitHash()}>Version {pkg.version} [Git Hash {getGitCommitHash()}]</a>
            </div>
        </div>
    </>
    {/*
    
    <>
        <div className="container">
            <div className="row">
                <div className="col l6 s12">
                    Made by <a className="orange-text text-lighten-3" href="https://materializeweb.com/">Materialize</a>
                </div>
                <div className="col l6 s12">
                    &copy; 2024 {pkg.author.name}
                </div>
            </div >
        </div >
        < div className="footer-copyright">
        </div >
    </>
    */}
}