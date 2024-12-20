import { useContext, useState } from "react";
import { apiContext } from "../Contexts";

function ObjectField(props: { value: any, id?: string, nkey?: string }) {
    //console.log("ObjectField got", props.nkey, props.value, typeof props.value)
    if (props.value === undefined || props.value === null || props.value === "") {
        return <></>
    } else if (typeof props.value === "string" || typeof props.value === "number") {
        return <tr key={"" + props.id + "-" + props.nkey}>
            <td>{props.nkey}</td>
            <td>{props.value}</td>
        </tr>
    } else if (typeof props.value === "boolean") {
        return <tr key={"" + props.id + "-" + props.nkey}>
            <td>{props.nkey}</td>
            <td>{props.value ? "Yes" : "No"}</td>
        </tr>
    } else if (Array.isArray(props.value)) {
        return props.value.map((item, index) => <ObjectField value={item} key={props.id + "." + index} id={props.id + "." + index} nkey={props.nkey} />)
    } else if (typeof props.value === "object") {
        return Object.getOwnPropertyNames(props.value).map((sel, index) => {
            //console.log("ObjectField", sel, Object.getOwnPropertyDescriptor(props.value, sel)?.value)
            return <ObjectField value={Object.getOwnPropertyDescriptor(props.value, sel)?.value} key={props.id + "." + index} id={props.id + "." + index} nkey={(props.nkey !== undefined ? props.nkey + "." : "") + sel} />
        })
    } else {
        return <tr key={"unk." + props.id}>
            <td>Unknown type: {typeof props.value}</td>
        </tr>;
    }
}

export function ObjectTable(props: { object: object | Array<any> | null | undefined }) {
    return <table>
        <thead>
            <tr>
                <th>Property</th>
                <th>Value</th>
            </tr>
        </thead>
        <tbody>
            {props.object ? <ObjectField value={props.object || {}} id={"0"} /> : <></>}
        </tbody>
    </table>
}