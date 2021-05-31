import * as msal from '@azure/msal-browser'
import React, { createContext, useEffect, useReducer } from 'react'
import Cookie from 'cookie-universal'

const msalConfig = {
    auth: {
        clientId: 'ef57b07c-de53-46f3-a8ce-976d7deba640',
        authority: 'https://login.microsoftonline.com/29eb35f9-68d5-4214-98d2-a6c0d92d5a75',
        redirectUri: 'http://localhost:3000/'
    },
    cache: {
        cacheLocation: "sessionStorage",
        storeAuthStateInCookie: false
    },
}

const authClient = new msal.PublicClientApplication(msalConfig)

const reducer = (state, action) => {
    switch (action.type) {
        case 'LOGGED_IN':
            return {
                ...state,
                isLoggedIn: true,
                working: {...state.working, int: false},
                user: action.user
            }
        case 'LOGGED_OUT':
            return {
                ...state,
                isLoggedIn: false,
                working: {...state.working, int: false},
                user: null
            }
        case 'START':
            return {
                ...state,
                working: {...state.working, int: true}
            }
        case 'SSTART':
            return {...state, working: {...state.working, sso: true}}
        case 'SLOGGED_IN':
            return {
                ...state,
                isLoggedIn: true,
                user: action.user,
                working: {...state.working, sso: false}
            }
        case 'SLOGGED_OUT':
            return {
                ...state,
                isLoggedIn: false,
                working: {...state.working, sso: false}
            }
        default:
            return state
    }
}

export const AuthContext = createContext()

const cookies = Cookie()

export const Auth = ({ children }) => {
    let [state, dispatch] = useReducer(reducer, { isLoggedIn: false, user: null, working: {sso: false, int: false} })

    const loginFunction = async () => {
        dispatch({ type: 'START' })
        try {
            await authClient.loginRedirect()
        } catch (e) {
        }
    }

    useEffect(() => {
        // if (document.location.href !== document.location.origin) {
        dispatch({ type: 'START' })
        // console.log("location redirectpromise", document.location)
        authClient.handleRedirectPromise().then((result) => {
            if (result == null) {
                console.log("no result in redirect!")
                dispatch({type: 'LOGGED_OUT'})
                return
            }
            cookies.set('u', result.account.username)
            dispatch({ type: 'LOGGED_IN', user: result.account })
        }).catch((e) => {
            // console.log("handleredirectpromise:", e)
            dispatch({ type: 'LOGGED_OUT' })
        })
        // }
    }, [])

    const logoutFunction = async () => {
        dispatch({ type: 'START' })
        await authClient.logout({ account: state.user })
        console.log("logoutFunction:")
        dispatch({ type: 'LOGGED_OUT' })
    }

    useEffect(() => {
        console.log("location sso", document.location)
        let u = cookies.get('u')
        if (u) {
            if (!document.location.href.includes("code=")) {
                dispatch({ type: 'SSTART' })
                authClient.ssoSilent({ loginHint: u })
                    .then((result) => dispatch({ type: 'SLOGGED_IN', user: result.account }))
                    .catch((e) => { /*console.log('sso:', e);*/ dispatch({ type: 'SLOGGED_OUT' }) })
            }
        }
    }, [])


    return (
        <AuthContext.Provider value={[{...state, working: (state.working.sso || state.working.int)}, loginFunction, logoutFunction]}>
            {children}
        </AuthContext.Provider>
    )
}
