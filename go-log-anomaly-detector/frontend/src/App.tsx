import React, { useEffect, useState } from 'react'
import AnomalyTable from './components/AnomalyTable'

export default function App() {
  const [rows, setRows] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  async function load() {
    try {
      const res = await fetch('/v1/anomalies')
      const data = await res.json()
      setRows(data)
    } finally { setLoading(false) }
  }

  useEffect(() => {
    load()
    const t = setInterval(load, 3000)
    return () => clearInterval(t)
  }, [])

  return (
    <div className="max-w-6xl mx-auto p-6 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Go Log Anomaly Detector</h1>
        <a href="/metrics" className="text-sm underline">/metrics</a>
      </header>

      {loading ? <p>Carregando…</p> : <AnomalyTable data={rows} />}

      <footer className="text-xs text-zinc-500 pt-6">
        Atualiza a cada 3s • <a className="underline" href="/healthz">/healthz</a>
      </footer>
    </div>
  )
}