// Or from '@reduxjs/toolkit/query' if not using the auto-generated hooks
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'

let APIURL = process.env.APIURL;
async function testURL(url: string): Promise<boolean> {
    try {
        const parsedUrl = new URL(url);
        return await fetch(url, { method: 'GET' })
            .then((response) => {
                if (response.ok) {
                    console.log(`API URL is reachable: ${url}`);
                    return true;
                } else {
                    console.error(`API URL is not reachable: ${url}`);
                    return false;
                }
            })
            .catch((error) => {
                console.error(`Error fetching API URL: ${error}`);
                return false;
            });
    } catch (e) {
        return false;
    }
}


// test if APIURL is set and if is reaceable
if (APIURL === undefined || APIURL === "") {
    console.error("APIURL is not set, using default: http://localhost:8080");
    APIURL = "http://localhost:8080";
} else if (process.env.APIURL === "dynamic" || !(await testURL(APIURL))) {
    APIURL = window.location.href.substring(0, window.location.href.lastIndexOf('/'));
    console.info(`Dynamic APIURL provided, using generated: ${APIURL}/ from ${window.location.href}`)
}
console.log("* API URL", APIURL + "/", "Reachable: ", await testURL(APIURL));

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