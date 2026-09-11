import React from 'react';
import type { DateRange } from './types';
import styles from '../ControlPlane.module.css';

export interface DateRangeFilterProps {
  value?: DateRange | null;
  onChange?: (range: DateRange | null) => void;
}

// Date-only values preserve the existing API contract and UTC day filtering.
export function DateRangeFilter({ value, onChange }: DateRangeFilterProps) {
  const change = (field: 'start' | 'end', date: string) => {
    if (!date) { onChange?.(null); return; }
    const start = field === 'start' ? date : value?.start || date;
    const end = field === 'end' ? date : value?.end || date;
    onChange?.({ start: start > end ? date : start, end: end < start ? date : end });
  };
  return (
    <div className={styles.dateRangeFilter} aria-label="Date range (UTC)">
      <label>From<input aria-label="From date (UTC)" type="date" value={value?.start || ''}
        disabled={!onChange} onChange={event => change('start', event.target.value)} /></label>
      <label>To<input aria-label="To date (UTC)" type="date" value={value?.end || ''}
        disabled={!onChange} onChange={event => change('end', event.target.value)} /></label>
      {value && <button type="button" className={styles.filterBtn} onClick={() => onChange?.(null)}>Clear dates</button>}
    </div>
  );
}
