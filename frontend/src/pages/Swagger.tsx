import { useState, useEffect, useRef } from "react";
import { useInView, InView } from "react-intersection-observer";
import { apiContext } from "../Contexts";
import SwaggerUI from "swagger-ui-react"
import "swagger-ui-react/swagger-ui.css"

export function Swagger() {
    return <InView as="div">
        <SwaggerUI url={apiContext.instance.getUri() + "/docs/swagger.json"} />
    </InView>

}