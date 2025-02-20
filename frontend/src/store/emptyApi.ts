// Or from '@reduxjs/toolkit/query' if not using the auto-generated hooks
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'

let APIURL = process.env.APIURL;
if (process.env.APIURL === "dynamic") {
    APIURL = window.location.href.substring(0, window.location.href.lastIndexOf('/static/') + 1);
    console.info(`Dynamic not APIURL provided, using generated: ${APIURL}/`)
}
console.log("* API URL", APIURL + "/");
// initialize an empty api service that we'll inject endpoints into later as needed
export const emptySplitApi = createApi({
    baseQuery: fetchBaseQuery({
        baseUrl: APIURL,
        prepareHeaders: (headers, { getState }) => {
            const token = /*getState().auth.token*/ "BOGUS_TOKEN" // FIXME: get the token from the store
            if (token) {
                headers.set('authorization', `Bearer ${token}`)
            }
            return headers
        }
    }),
    endpoints: () => ({}),
})

export const apiUrl = APIURL + "/";