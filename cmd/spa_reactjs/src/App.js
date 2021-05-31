import React, { useContext, useEffect } from 'react';
import { AuthContext, Auth } from './auth'

export default () => (
  <>
    <Auth>
      <LoginLogoutExample />
    </Auth>
  </>
);

// Fun with reactjs components
const LoginLogoutExample = () => {
  let [auth, login, logout] = useContext(AuthContext)
  useEffect(() => console.log("auth", auth))
  if (auth.working) {
    return <h1>Authenticating...</h1>
  }
  if (auth.isLoggedIn) {
    return (
      <>
        <User user={auth.user} />
        <button onClick={logout}>Logout</button>
      </>
    )
  }
  return <button onClick={login}>Login with your CPA Account</button>
}

const User = ({ user }) => {
  if (user == null) {
    return "User object not found!"
  }

  return (
    <table>
      <tbody>
        <tr>
          <th>Key</th>
          <th>Value</th>
        </tr>
        <tr>
          <th>homeAccountId</th><td>{user.homeAccountId}</td>
        </tr>
        <tr>
          <th>environment</th><td>{user.environment}</td>
        </tr>
        <tr>
          <th>tenantId</th><td>{user.tenantId}</td>
        </tr>
        <tr>
          <th>username</th><td>{user.username}</td>
        </tr>
      </tbody>
    </table>
  )
}


