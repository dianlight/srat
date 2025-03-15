import { InView } from "react-intersection-observer";
//import { apiContext } from "../Contexts";
import SwaggerUI from "swagger-ui-react"
import "swagger-ui-react/swagger-ui.css"
import { apiUrl } from "../store/emptyApi";

export function Swagger() {

    return <InView as="div">
        <SwaggerUI url={apiUrl + "openapi-3.0.json"} />
    </InView>

}