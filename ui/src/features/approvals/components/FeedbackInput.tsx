/**
 * FeedbackInput.tsx - Feedback textarea with character counter
 *
 * Used in rejection modal to collect feedback with max length validation.
 * Part of M-DASHBOARD-APPROVAL-INTEGRATION multi-channel workflow.
 */

import React, { useCallback } from 'react';
import './FeedbackInput.css';

interface FeedbackInputProps {
  value: string;
  onChange: (value: string) => void;
  maxLength?: number;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  autoFocus?: boolean;
  rows?: number;
  label?: string;
  error?: string;
}

export const FeedbackInput: React.FC<FeedbackInputProps> = ({
  value,
  onChange,
  maxLength = 1000,
  placeholder = 'Please explain why this change is being rejected...',
  required = true,
  disabled = false,
  autoFocus = true,
  rows = 4,
  label = 'Rejection feedback',
  error,
}) => {
  const charCount = value.length;
  const isNearLimit = charCount > maxLength * 0.8;
  const isOverLimit = charCount > maxLength;

  const handleChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    // Allow typing but warn if over limit
    onChange(newValue);
  }, [onChange]);

  return (
    <div className={`feedback-input-container ${error ? 'has-error' : ''}`}>
      {label && (
        <label className="feedback-label">
          {label}
          {required && <span className="required-marker">*</span>}
        </label>
      )}

      <textarea
        className={`feedback-textarea ${isOverLimit ? 'over-limit' : ''}`}
        value={value}
        onChange={handleChange}
        placeholder={placeholder}
        disabled={disabled}
        autoFocus={autoFocus}
        rows={rows}
        maxLength={maxLength + 100} // Allow slight overflow for warning
      />

      <div className="feedback-footer">
        {error && (
          <span className="feedback-error">{error}</span>
        )}
        <span className={`char-counter ${isNearLimit ? 'near-limit' : ''} ${isOverLimit ? 'over-limit' : ''}`}>
          {charCount}/{maxLength}
        </span>
      </div>

      {isOverLimit && (
        <div className="over-limit-warning">
          Feedback is too long. Please reduce by {charCount - maxLength} characters.
        </div>
      )}
    </div>
  );
};

export default FeedbackInput;
