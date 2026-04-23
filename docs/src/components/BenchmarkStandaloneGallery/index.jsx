import React, { useState, useEffect } from 'react';
import BenchmarkGallery from '../BenchmarkDashboard/BenchmarkGallery';

export default function BenchmarkStandaloneGallery() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch('/benchmarks/latest.json')
      .then(r => r.json())
      .then(setData)
      .catch(e => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load benchmarks: {error}</p>;
  if (!data) return <p>Loading benchmarks…</p>;
  if (!data.benchmarks) return <p>No benchmark data found.</p>;

  return <BenchmarkGallery benchmarks={data.benchmarks} />;
}
