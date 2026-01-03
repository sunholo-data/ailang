import React, { useState, useEffect } from 'react';
import './RunningTasks.css';

interface Task {
  id: string;
  title: string;
  status: string;
  type: string;
  provider?: string;
  created_at: string;
  started_at?: string;
  thread_id?: string;
}

interface RunningTasksProps {
  onSelectTask: (taskId: string, threadId?: string) => void;
}

export const RunningTasks: React.FC<RunningTasksProps> = ({ onSelectTask }) => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTasks = async () => {
      try {
        const response = await fetch('/api/coordinator/status');
        if (!response.ok) {
          throw new Error('Failed to fetch coordinator status');
        }
        const data = await response.json();
        // The status endpoint returns task stats, but we need the actual tasks
        // Let's also fetch from the tasks endpoint if available
        setTasks(data.running_tasks || []);
        setLoading(false);
      } catch (err) {
        console.error('Failed to fetch running tasks:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
        setLoading(false);
      }
    };

    fetchTasks();
    // Poll for updates every 5 seconds
    const interval = setInterval(fetchTasks, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="running-tasks">
        <h3>Running Tasks</h3>
        <div className="loading">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="running-tasks">
        <h3>Running Tasks</h3>
        <div className="error">{error}</div>
      </div>
    );
  }

  if (tasks.length === 0) {
    return (
      <div className="running-tasks">
        <h3>Running Tasks</h3>
        <div className="no-tasks">No tasks currently running</div>
      </div>
    );
  }

  return (
    <div className="running-tasks">
      <h3>Running Tasks ({tasks.length})</h3>
      <div className="task-list">
        {tasks.map((task) => (
          <div
            key={task.id}
            className={`task-card task-${task.status}`}
            onClick={() => onSelectTask(task.id, task.thread_id)}
          >
            <div className="task-header">
              <span className="task-title">{task.title || task.id}</span>
              <span className={`task-status status-${task.status}`}>
                {task.status}
              </span>
            </div>
            <div className="task-meta">
              <span className="task-type">{task.type}</span>
              {task.provider && <span className="task-provider">{task.provider}</span>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default RunningTasks;
