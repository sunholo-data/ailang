/**
 * Message input area with workspace selector and kind dropdown
 */
import React, { useState } from 'react';
import { Icons } from '../common/Icons';

interface MessageInputProps {
  workspace: string;
  onWorkspaceChange: (workspace: string) => void;
  onSendMessage: (content: string, kind: string, workspace?: string) => void;
}

export const MessageInput: React.FC<MessageInputProps> = ({
  workspace,
  onWorkspaceChange,
  onSendMessage,
}) => {
  const [inputValue, setInputValue] = useState('');
  const [messageKind, setMessageKind] = useState<string>('directive');
  const [showWorkspaceInput, setShowWorkspaceInput] = useState<boolean>(false);

  const handleSend = () => {
    if (inputValue.trim()) {
      onSendMessage(inputValue, messageKind, workspace || undefined);
      setInputValue('');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleBrowseFolder = async () => {
    try {
      const response = await fetch('/api/select-folder');
      const data = await response.json();
      if (!data.cancelled && data.path) {
        onWorkspaceChange(data.path);
      }
    } catch (err) {
      console.error('Failed to open folder picker:', err);
    }
  };

  return (
    <div className="input-area">
      {/* Workspace expanded input row */}
      {showWorkspaceInput && (
        <div className="workspace-input-row">
          <input
            type="text"
            value={workspace}
            onChange={(e) => onWorkspaceChange(e.target.value)}
            placeholder="/path/to/working/directory (leave empty for fresh workspace)"
            className="workspace-input"
          />
          <button
            onClick={handleBrowseFolder}
            className="workspace-browse"
            title="Browse for folder"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
              <line x1="12" y1="11" x2="12" y2="17" />
              <line x1="9" y1="14" x2="15" y2="14" />
            </svg>
          </button>
          {workspace && (
            <button
              onClick={() => { onWorkspaceChange(''); setShowWorkspaceInput(false); }}
              className="workspace-clear"
            >
              Clear
            </button>
          )}
        </div>
      )}

      <div className="input-wrapper">
        {/* Workspace button on the LEFT */}
        <button
          onClick={() => setShowWorkspaceInput(!showWorkspaceInput)}
          className={`workspace-toggle ${workspace ? 'has-workspace' : ''}`}
          title={workspace || 'Set working directory for agent tasks'}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
          </svg>
        </button>
        <select
          value={messageKind}
          onChange={(e) => setMessageKind(e.target.value)}
          className="kind-selector"
          title={messageKind === 'directive'
            ? 'Directive: A task or instruction for the agent to execute'
            : 'Question: A query for information (won\'t trigger execution)'}
        >
          <option value="directive" title="A task or instruction for the agent to execute">Directive</option>
          <option value="question" title="A query for information (won't trigger execution)">Question</option>
        </select>
        <textarea
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder={workspace ? `Message (workspace: ${workspace.split('/').pop()})` : "Type a message..."}
          rows={1}
        />
        <button
          onClick={handleSend}
          className="send-btn"
          disabled={!inputValue.trim()}
        >
          {Icons.send}
        </button>
      </div>
      <div className="input-hint">
        Press <kbd>Enter</kbd> to send, <kbd>Shift + Enter</kbd> for new line
      </div>
    </div>
  );
};
