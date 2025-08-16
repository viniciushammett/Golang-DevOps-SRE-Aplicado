import React from 'react'

type Anomaly = {
  id: string
  when: string
  rule: string
  kind: string
  sample: string
  count: number
  window: string
  severity: string
}

export default function AnomalyTable({ data }: { data: Anomaly[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-sm">
        <thead className="text-zinc-400 uppercase">
          <tr>
            <th className="p-2">Quando</th>
            <th className="p-2">Regra</th>
            <th className="p-2">Tipo</th>
            <th className="p-2">Count</th>
            <th className="p-2">Sev</th>
            <th className="p-2">Amostra</th>
          </tr>
        </thead>
        <tbody>
          {data.map(a => (
            <tr key={a.id} className="border-b border-zinc-800">
              <td className="p-2">{new Date(a.when).toLocaleString()}</td>
              <td className="p-2 font-medium">{a.rule}</td>
              <td className="p-2">{a.kind}</td>
              <td className="p-2">{a.count}</td>
              <td className="p-2">{a.severity || '-'}</td>
              <td className="p-2"><code className="text-zinc-300">{a.sample}</code></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}