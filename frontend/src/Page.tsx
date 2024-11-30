import { Shares } from "./Shares";

export function Page(/*props: { message: string }*/) {
    return <div className="row">
        <div className="col s12">
            <ul className="tabs">
                <li className="tab col s4"><a className="active" href="#shares">Shares</a></li>
                <li className="tab col s4"><a href="#users">Users</a></li>
                {/*<li className="tab col s3 disabled"><a href="#test3">Disabled Tab</a></li>*/}
                <li className="tab col s4 disabled"><a href="#settings">Settings</a></li>
            </ul>
        </div>
        <div id="shares" className="col s12"><Shares /></div>
        <div id="users" className="col s12">Users</div>
        {/*<div id="test3" className="col s12">Test 3</div>*/}
        <div id="settings" className="col s12">Settings</div>
        {/*props.message*/}
    </div>
}