/**
 * Inline approval panel for approval_request messages
 */
import React from 'react';
import { Icons } from '../common/Icons';

interface InlineApprovalPanelProps {
  approvalId: string;
  isHandled: boolean;
  notes: string;
  onNotesChange: (notes: string) => void;
  onApprove: () => void;
  onReject: () => void;
}

export const InlineApprovalPanel: React.FC<InlineApprovalPanelProps> = ({
  isHandled,
  notes,
  onNotesChange,
  onApprove,
  onReject,
}) => {
  if (isHandled) {
    return (
      <div className="inline-approval">
        <div className="approval-handled">
          {Icons.check}
          <span>Action taken</span>
        </div>
      </div>
    );
  }

  return (
    <div className="inline-approval">
      <input
        type="text"
        className="approval-notes-input"
        placeholder="Notes (required for rejection)..."
        value={notes}
        onChange={(e) => onNotesChange(e.target.value)}
      />
      <div className="approval-actions">
        <button
          className="reject-btn"
          onClick={onReject}
          title="Reject"
        >
          {Icons.x}
          Reject
        </button>
        <button
          className="approve-btn"
          onClick={onApprove}
          title="Approve"
        >
          {Icons.check}
          Approve
        </button>
      </div>
    </div>
  );
};
