import React, { useState, useEffect } from 'react'

const API = import.meta.env.VITE_GSV_ADDR || 'http://localhost:8080'

export default function App(){
  const [token,setToken] = useState(localStorage.getItem('gsv_token')||'')
  const [user,setUser] = useState('admin')
  const [pass,setPass] = useState('admin')
  const [list,setList] = useState([])
  const [name,setName] = useState('example.key')
  const [val,setVal] = useState('value')
  const [ttl,setTtl] = useState('1h')

  const auth = token? { 'Authorization': `Bearer ${token}` } : {}

  async function login(){
    const r = await fetch(`${API}/login`, {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({username:user,password:pass})})
    const j = await r.json(); setToken(j.token); localStorage.setItem('gsv_token', j.token)
    load()
  }
  async function load(){
    if(!token) return; const r = await fetch(`${API}/secrets/`, {headers: auth}); setList(await r.json())
  }
  async function create(){
    await fetch(`${API}/secrets/`, {method:'POST', headers:{...auth,'Content-Type':'application/json'}, body: JSON.stringify({name, value:val, ttl})});
    load()
  }
  useEffect(()=>{ load() }, [token])

  if(!token) return (
    <div style={{maxWidth:420, margin:'40px auto', fontFamily:'system-ui'}}>
      <h1>Go Secret Vault</h1>
      <input value={user} onChange={e=>setUser(e.target.value)} placeholder='user' />
      <input type='password' value={pass} onChange={e=>setPass(e.target.value)} placeholder='password' />
      <button onClick={login}>Login</button>
    </div>
  )

  return (
    <div style={{maxWidth:800, margin:'40px auto', fontFamily:'system-ui'}}>
      <h1>Secrets</h1>
      <div style={{display:'flex', gap:8}}>
        <input value={name} onChange={e=>setName(e.target.value)} placeholder='name' style={{flex:1}}/>
        <input value={val} onChange={e=>setVal(e.target.value)} placeholder='value' style={{flex:1}}/>
        <input value={ttl} onChange={e=>setTtl(e.target.value)} placeholder='ttl e.g. 1h' style={{width:120}}/>
        <button onClick={create}>Create</button>
      </div>
      <ul>
        {list.map(s=> <li key={s.id}><strong>{s.name}</strong> — id: {s.id} {s.expires_at? `(exp: ${s.expires_at})`:''}</li>)}
      </ul>
      <button onClick={()=>{ localStorage.removeItem('gsv_token'); setToken('') }}>Logout</button>
    </div>
  )
}