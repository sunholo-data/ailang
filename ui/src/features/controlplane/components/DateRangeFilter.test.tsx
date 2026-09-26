import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { DateRangeFilter } from './DateRangeFilter';
import { MessageQueue } from './MessageQueue';

// Exercise the actual controlled inputs' event handlers without a browser or
// production requests. The parent still owns the API date-only filter values.
function controls(value: { start: string; end: string } | null = null) {
  const onChange = vi.fn();
  const element = DateRangeFilter({ value, onChange });
  const labels = element.props.children.slice(0, 2);
  return { onChange, element, inputs: labels.map((label: React.ReactElement) => label.props.children[1]) };
}

describe('direct date filters', () => {
  it('offers usable date inputs before a date range is selected', () => {
    const { element, inputs, onChange } = controls();
    const markup = renderToStaticMarkup(element);
    expect(markup).toContain('From date (UTC)');
    expect(markup).toContain('To date (UTC)');
    expect(markup).not.toContain('disabled');
    inputs[0].props.onChange({ target: { value: '2026-09-08' } });
    expect(onChange).toHaveBeenCalledWith({ start: '2026-09-08', end: '2026-09-08' });
  });
  it('extends a selected day without converting the date-only API values', () => {
    const { inputs, onChange } = controls({ start: '2026-09-01', end: '2026-09-01' });
    inputs[1].props.onChange({ target: { value: '2026-09-08' } });
    expect(onChange).toHaveBeenCalledWith({ start: '2026-09-01', end: '2026-09-08' });
  });
  it('keeps the range ordered when moving either boundary past the other', () => {
    const { inputs, onChange } = controls({ start: '2026-09-02', end: '2026-09-08' });
    inputs[0].props.onChange({ target: { value: '2026-09-10' } });
    expect(onChange).toHaveBeenLastCalledWith({ start: '2026-09-10', end: '2026-09-10' });
    inputs[1].props.onChange({ target: { value: '2026-09-01' } });
    expect(onChange).toHaveBeenLastCalledWith({ start: '2026-09-01', end: '2026-09-01' });
  });
  it('clears the parent filter from either the field or Clear dates button', () => {
    const { element, inputs, onChange } = controls({ start: '2026-09-01', end: '2026-09-08' });
    inputs[0].props.onChange({ target: { value: '' } });
    expect(onChange).toHaveBeenLastCalledWith(null);
    element.props.children[2].props.onClick();
    expect(onChange).toHaveBeenLastCalledWith(null);
  });
});

// This renders the retained queue, catching disagreement between date controls
// and the event filter (including the final millisecond of a UTC day).
it('keeps UTC boundary events and excludes adjacent days in the queue', () => {
  const events = [
    ['2026-09-07T23:59:59.999Z', 'outside-before'],
    ['2026-09-08T00:00:00.000Z', 'inside-start'],
    ['2026-09-08T23:59:59.999Z', 'inside-end'],
    ['2026-09-09T00:00:00.000Z', 'outside-after'],
  ].map(([timestamp, content]) => ({ id: content, timestamp, content, type: 'message' as const, source: 'fixture' }));
  const markup = renderToStaticMarkup(<MessageQueue events={events} onEventClick={() => {}}
    selectedDateRange={{ start: '2026-09-08', end: '2026-09-08' }} onDateRangeChange={() => {}} />);
  expect(markup).toContain('inside-start');
  expect(markup).toContain('inside-end');
  expect(markup).not.toContain('outside-before');
  expect(markup).not.toContain('outside-after');
});
