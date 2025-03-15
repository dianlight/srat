// Or from '@reduxjs/toolkit/query' if not using the auto-generated hooks
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'

let APIURL = process.env.APIURL;
if (process.env.APIURL === "dynamic") {
    APIURL = window.location.href.substring(0, window.location.href.lastIndexOf('/'));
    console.info(`Dynamic APIURL provided, using generated: ${APIURL}/ from ${window.location.href}`)
}
console.log("* API URL", APIURL + "/");
// initialize an empty api service that we'll inject endpoints into later as needed
export const emptySplitApi = createApi({
    baseQuery: fetchBaseQuery({
        baseUrl: APIURL,
        /*
          HA use auto-generated headers see https://developers.home-assistant.io/docs/add-ons/security#authenticating-a-user-when-using-ingress
          this can be implemented to allow UI outside HA with authentication

        prepareHeaders: (headers, { getState }) => {
            const token = getState().auth.token 
            if (token) {
                headers.set('authorization', `Bearer ${token}`)
            }
            return headers
    }
        */
    }),
    endpoints: () => ({}),
})

export const apiUrl = APIURL + "/";