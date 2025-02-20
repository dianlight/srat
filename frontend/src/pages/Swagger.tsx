import { useState, useEffect, useRef } from "react";
import { useInView, InView } from "react-intersection-observer";
//import { apiContext } from "../Contexts";
import SwaggerUI from "swagger-ui-react"
import "swagger-ui-react/swagger-ui.css"
import { sratApi } from "../store/sratApi";
import { apiUrl, emptySplitApi } from "../store/emptyApi";

export function Swagger() {

    return <InView as="div">
        <SwaggerUI url={apiUrl + "docs/swagger.json"} />
    </InView>

}