import React, { useState, useEffect } from 'react';
import { benchmarkFetchWithSource } from '@site/src/lib/benchmarkFetch';
import BenchmarkGallery from '../BenchmarkDashboard/BenchmarkGallery';
import DataProvenance from '../DataProvenance';

export default function BenchmarkStandaloneGallery() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [dataSource, setDataSource] = useState(null);

  useEffect(() => {
    benchmarkFetchWithSource('latest.json')
      .then(({ response, source }) => {
        setDataSource(source);
        return response.json();
      })
      .then(setData)
      .catch(e => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load benchmarks: {error}</p>;
  if (!data) return <p>Loading benchmarks…</p>;
  if (!data.benchmarks) return <p>No benchmark data found.</p>;

  return (
    <div>
      <DataProvenance version={data.version} timestamp={data.timestamp} source={dataSource} />
      <BenchmarkGallery benchmarks={data.benchmarks} ratings={data.ratings} />
    </div>
  );
}
