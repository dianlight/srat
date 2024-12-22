import { useForm } from "react-hook-form";
import { DevTool } from "@hookform/devtools";
import { useContext, useRef } from "react";
import { ModeContext } from "../Contexts";

export function Settings() {
    const { register, control, handleSubmit } = useForm({
        mode: "onChange",
    });
    const mode = useContext(ModeContext);

    const formRef = useRef<HTMLFormElement>(null);

    return (
        <>
            <div className="card">
                <div className="card-content">
                    <form id="settingsform" ref={formRef} className="row" onSubmit={handleSubmit(d => console.log(d))}>
                        <div className="s12 m6 input-field inline">
                            <input id="workgroup" type="text" className="validate" placeholder=" "  {...register("workgroup", { required: true })} readOnly={mode.read_only} />
                            <label htmlFor="workgroup">Workgroup</label>
                        </div>
                        <div className="s12 m6">
                            <p className="caption">Allow Host</p>
                            <div className="chips chips-placeholder"></div>
                        </div>
                    </form>
                </div>
                <div className="card-action">
                    <a href="#!" className="modal-close waves-effect btn-flat">Disagree</a>
                    <input type="submit" form="settingsform" className="modal-close waves-effect btn-flat" value="Agree" />
                </div>
            </div>

            <DevTool control={control} /> {/* set up the dev tool */}
        </>
    );
}